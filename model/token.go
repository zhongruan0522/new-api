package model

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/go-redis/redis/v8"
	"github.com/zhongruan0522/new-api/common"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"
)

// maxUserTokens 每用户最大令牌数量（硬编码）
const maxUserTokens = 1000

// tokenLoadGroup 合并并发的冷缓存 token 查询，避免启动瞬间把相同 key 打到数据库上。
var tokenLoadGroup singleflight.Group

// tokenResetLocks 令牌密钥重置互斥锁，按 token ID 串行化并发重置请求，
// 防止同一令牌同时出现多个有效 key。
var tokenResetLocks sync.Map

func getTokenResetLock(id int) *sync.Mutex {
	v, _ := tokenResetLocks.LoadOrStore(id, &sync.Mutex{})
	return v.(*sync.Mutex)
}

type Token struct {
	Id                 int            `json:"id"`
	UserId             int            `json:"user_id" gorm:"index"`
	Key                string         `json:"key" gorm:"type:char(48);uniqueIndex"`
	Status             int            `json:"status" gorm:"default:1"`
	Name               string         `json:"name" gorm:"index" `
	CreatedTime        int64          `json:"created_time" gorm:"bigint"`
	AccessedTime       int64          `json:"accessed_time" gorm:"bigint"`
	ExpiredTime        int64          `json:"expired_time" gorm:"bigint;default:-1"` // -1 means never expired
	RemainQuota        int            `json:"remain_quota" gorm:"default:0"`
	UnlimitedQuota     bool           `json:"unlimited_quota"`
	ModelLimitsEnabled bool           `json:"model_limits_enabled"`
	ModelLimits        string         `json:"model_limits" gorm:"type:varchar(1024);default:''"`
	AllowIps           *string        `json:"allow_ips" gorm:"default:''"`
	UsedQuota          int            `json:"used_quota" gorm:"default:0"` // used quota
	Group              string         `json:"group" gorm:"default:''"`
	CrossGroupRetry    bool           `json:"cross_group_retry"` // 跨分组重试，仅auto分组有效

	// 限额类型：0=无限额度, 1=永久限额, 2=时段限额, 3=时段+周期限额
	QuotaType int `json:"quota_type" gorm:"default:0"`

	// 时段限额相关字段（quota_type=2,3 时生效）
	WindowHours     int `json:"window_hours" gorm:"default:0"`      // 窗口时长（小时）
	WindowQuota     int `json:"window_quota" gorm:"default:0"`      // 每个窗口的额度
	WindowStartHour int `json:"window_start_hour" gorm:"default:0"` // 窗口起始小时（0-23）

	// 周期限额相关字段（quota_type=3 时生效）
	CycleDays  int `json:"cycle_days" gorm:"default:0"`  // 周期天数
	CycleQuota int `json:"cycle_quota" gorm:"default:0"` // 周期总额度

	// 运行时状态字段（自动计算，不由用户设置）
	WindowUsedQuota int   `json:"window_used_quota" gorm:"default:0"` // 当前窗口已用额度
	WindowStartTime int64 `json:"window_start_time" gorm:"default:0"` // 当前窗口开始时间（unix timestamp）
	CycleUsedQuota  int   `json:"cycle_used_quota" gorm:"default:0"`  // 当前周期已用额度
	CycleStartTime  int64 `json:"cycle_start_time" gorm:"default:0"`  // 当前周期开始时间（unix timestamp）

	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (token *Token) Clean() {
	token.Key = ""
}

func (token *Token) GetIpLimits() []string {
	// delete empty spaces
	//split with \n
	ipLimits := make([]string, 0)
	if token.AllowIps == nil {
		return ipLimits
	}
	cleanIps := strings.ReplaceAll(*token.AllowIps, " ", "")
	if cleanIps == "" {
		return ipLimits
	}
	ips := strings.Split(cleanIps, "\n")
	for _, ip := range ips {
		ip = strings.TrimSpace(ip)
		ip = strings.ReplaceAll(ip, ",", "")
		if ip != "" {
			ipLimits = append(ipLimits, ip)
		}
	}
	return ipLimits
}

func GetAllUserTokens(userId int, startIdx int, num int) ([]*Token, error) {
	var tokens []*Token
	var err error
	err = DB.Where("user_id = ?", userId).Order("id desc").Limit(num).Offset(startIdx).Find(&tokens).Error
	return tokens, err
}

// sanitizeLikePattern 校验并清洗用户输入的 LIKE 搜索模式。
// 规则：
//  1. 转义 ! 和 _（使用 ! 作为 ESCAPE 字符，兼容 MySQL/PostgreSQL/SQLite）
//  2. 连续的 % 合并为单个 %
//  3. 最多允许 2 个 %
//  4. 含 % 时（模糊搜索），去掉 % 后关键词长度必须 >= 2
//  5. 不含 % 时按精确匹配
func sanitizeLikePattern(input string) (string, error) {
	// 1. 先转义 ESCAPE 字符 ! 自身，再转义 _
	//    使用 ! 而非 \ 作为 ESCAPE 字符，避免 MySQL 中反斜杠的字符串转义问题
	input = strings.ReplaceAll(input, "!", "!!")
	input = strings.ReplaceAll(input, `_`, `!_`)

	// 2. 连续的 % 直接拒绝
	if strings.Contains(input, "%%") {
		return "", errors.New("搜索模式中不允许包含连续的 % 通配符")
	}

	// 3. 统计 % 数量，不得超过 2
	count := strings.Count(input, "%")
	if count > 2 {
		return "", errors.New("搜索模式中最多允许包含 2 个 % 通配符")
	}

	// 4. 含 % 时，去掉 % 后关键词长度必须 >= 2
	if count > 0 {
		stripped := strings.ReplaceAll(input, "%", "")
		if len(stripped) < 2 {
			return "", errors.New("使用模糊搜索时，关键词长度至少为 2 个字符")
		}
		return input, nil
	}

	// 5. 无 % 时，精确全匹配
	return input, nil
}

const searchHardLimit = 100

func SearchUserTokens(userId int, keyword string, token string, offset int, limit int) (tokens []*Token, total int64, err error) {
	// model 层强制截断
	if limit <= 0 || limit > searchHardLimit {
		limit = searchHardLimit
	}
	if offset < 0 {
		offset = 0
	}

	if token != "" {
		token = strings.Trim(token, "sk-")
	}

	// 超量用户（令牌数超过上限）只允许精确搜索，禁止模糊搜索
	hasFuzzy := strings.Contains(keyword, "%") || strings.Contains(token, "%")
	if hasFuzzy {
		count, err := CountUserTokens(userId)
		if err != nil {
			common.SysLog("failed to count user tokens: " + err.Error())
			return nil, 0, errors.New("获取令牌数量失败")
		}
		if int(count) > maxUserTokens {
			return nil, 0, errors.New("令牌数量超过上限，仅允许精确搜索，请勿使用 % 通配符")
		}
	}

	baseQuery := DB.Model(&Token{}).Where("user_id = ?", userId)

	// 非空才加 LIKE 条件，空则跳过（不过滤该字段）
	if keyword != "" {
		keywordPattern, err := sanitizeLikePattern(keyword)
		if err != nil {
			return nil, 0, err
		}
		baseQuery = baseQuery.Where("name LIKE ? ESCAPE '!'", keywordPattern)
	}
	if token != "" {
		tokenPattern, err := sanitizeLikePattern(token)
		if err != nil {
			return nil, 0, err
		}
		baseQuery = baseQuery.Where(commonKeyCol+" LIKE ? ESCAPE '!'", tokenPattern)
	}

	// 先查匹配总数（用于分页，受 maxTokens 上限保护，避免全表 COUNT）
	err = baseQuery.Limit(maxUserTokens).Count(&total).Error
	if err != nil {
		common.SysError("failed to count search tokens: " + err.Error())
		return nil, 0, errors.New("搜索令牌失败")
	}

	// 再分页查数据
	err = baseQuery.Order("id desc").Offset(offset).Limit(limit).Find(&tokens).Error
	if err != nil {
		common.SysError("failed to search tokens: " + err.Error())
		return nil, 0, errors.New("搜索令牌失败")
	}
	return tokens, total, nil
}

func ValidateUserToken(key string) (token *Token, err error) {
	if key == "" {
		return nil, errors.New("未提供令牌")
	}
	token, err = GetTokenByKey(key, false)
	if err == nil {
		if token.Status == common.TokenStatusExhausted {
			keyPrefix := key[:3]
			keySuffix := key[len(key)-3:]
			return token, errors.New("该令牌额度已用尽 TokenStatusExhausted[sk-" + keyPrefix + "***" + keySuffix + "]")
		} else if token.Status == common.TokenStatusExpired {
			return token, errors.New("该令牌已过期")
		}
		if token.Status != common.TokenStatusEnabled {
			return token, errors.New("该令牌状态不可用")
		}
		if token.ExpiredTime != -1 && token.ExpiredTime < common.GetTimestamp() {
			if !common.RedisEnabled {
				token.Status = common.TokenStatusExpired
				err := token.SelectUpdate()
				if err != nil {
					common.SysLog("failed to update token status" + err.Error())
				}
			}
			return token, errors.New("该令牌已过期")
		}

		// 兼容旧数据：如果没有 QuotaType，从 UnlimitedQuota 派生
		quotaType := token.QuotaType
		if quotaType == 0 && !token.UnlimitedQuota {
			quotaType = 1
		}

		switch quotaType {
		case 0: // 无限额度
			// 不做额度检查
		case 1: // 永久限额
			if token.RemainQuota <= 0 {
				if !common.RedisEnabled {
					token.Status = common.TokenStatusExhausted
					err := token.SelectUpdate()
					if err != nil {
						common.SysLog("failed to update token status" + err.Error())
					}
				}
				keyPrefix := key[:3]
				keySuffix := key[len(key)-3:]
				return token, errors.New(fmt.Sprintf("[sk-%s***%s] 该令牌额度已用尽 !token.UnlimitedQuota && token.RemainQuota = %d", keyPrefix, keySuffix, token.RemainQuota))
			}
		case 2: // 时段限额
			if token.ShouldResetWindow() {
				windowStart, _ := token.GetCurrentWindow()
				if err := ResetWindowQuota(token.Id, token.WindowStartTime, windowStart); err != nil {
					// CAS 竞争失败或 DB 错误，尝试重新加载最新 token
					if fresh, loadErr := GetTokenByKey(key, true); loadErr == nil && fresh != nil {
						token = fresh
					} else {
						common.SysLog("failed to reset window quota: " + err.Error())
						return token, errors.New("令牌窗口状态更新失败，请重试")
					}
				} else {
					token.WindowUsedQuota = 0
					token.WindowStartTime = windowStart
					_ = cacheDeleteToken(token.Key)
				}
			}
			if token.WindowUsedQuota >= token.WindowQuota {
				keyPrefix := key[:3]
				keySuffix := key[len(key)-3:]
				return token, errors.New(fmt.Sprintf("[sk-%s***%s] 该令牌时段额度已用尽 (窗口已用: %d, 窗口总额: %d)", keyPrefix, keySuffix, token.WindowUsedQuota, token.WindowQuota))
			}
		case 3: // 时段+周期限额
			if token.ShouldResetWindow() {
				windowStart, _ := token.GetCurrentWindow()
				if err := ResetWindowQuota(token.Id, token.WindowStartTime, windowStart); err != nil {
					if fresh, loadErr := GetTokenByKey(key, true); loadErr == nil && fresh != nil {
						token = fresh
					} else {
						common.SysLog("failed to reset window quota: " + err.Error())
						return token, errors.New("令牌窗口状态更新失败，请重试")
					}
				} else {
					token.WindowUsedQuota = 0
					token.WindowStartTime = windowStart
					_ = cacheDeleteToken(token.Key)
				}
			}
			if token.ShouldResetCycle() {
				cycleStart, _ := token.GetCurrentCycle()
				if err := ResetCycleQuota(token.Id, token.CycleStartTime, cycleStart); err != nil {
					if fresh, loadErr := GetTokenByKey(key, true); loadErr == nil && fresh != nil {
						token = fresh
					} else {
						common.SysLog("failed to reset cycle quota: " + err.Error())
						return token, errors.New("令牌周期状态更新失败，请重试")
					}
				} else {
					token.CycleUsedQuota = 0
					token.CycleStartTime = cycleStart
					_ = cacheDeleteToken(token.Key)
				}
			}
			if token.WindowUsedQuota >= token.WindowQuota {
				keyPrefix := key[:3]
				keySuffix := key[len(key)-3:]
				return token, errors.New(fmt.Sprintf("[sk-%s***%s] 该令牌时段额度已用尽 (窗口已用: %d, 窗口总额: %d)", keyPrefix, keySuffix, token.WindowUsedQuota, token.WindowQuota))
			}
			if token.CycleUsedQuota >= token.CycleQuota {
				keyPrefix := key[:3]
				keySuffix := key[len(key)-3:]
				return token, errors.New(fmt.Sprintf("[sk-%s***%s] 该令牌周期额度已用尽 (周期已用: %d, 周期总额: %d)", keyPrefix, keySuffix, token.CycleUsedQuota, token.CycleQuota))
			}
		}
		return token, nil
	}
	common.SysLog("ValidateUserToken: failed to get token: " + err.Error())
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("无效的令牌")
	} else {
		return nil, errors.New("无效的令牌，数据库查询出错，请联系管理员")
	}
}

func GetTokenByIds(id int, userId int) (*Token, error) {
	if id == 0 || userId == 0 {
		return nil, errors.New("id 或 userId 为空！")
	}
	token := Token{Id: id, UserId: userId}
	var err error = nil
	err = DB.First(&token, "id = ? and user_id = ?", id, userId).Error
	return &token, err
}

func GetTokenById(id int) (*Token, error) {
	if id == 0 {
		return nil, errors.New("id 为空！")
	}
	token := Token{Id: id}
	var err error = nil
	err = DB.First(&token, "id = ?", id).Error
	if shouldUpdateRedis(true, err) {
		gopool.Go(func() {
			if err := cacheSetToken(token); err != nil {
				common.SysLog("failed to update user status cache: " + err.Error())
			}
		})
	}
	return &token, err
}

func GetTokenByKey(key string, fromDB bool) (token *Token, err error) {
	defer func() {
		// Update Redis cache asynchronously on successful DB read
		if shouldUpdateRedis(fromDB, err) && token != nil {
			gopool.Go(func() {
				if err := cacheSetToken(*token); err != nil {
					common.SysLog("failed to update user status cache: " + err.Error())
				}
			})
		}
	}()
	if !fromDB && common.RedisEnabled {
		// Try Redis first
		token, err := cacheGetTokenByKey(key)
		if err == nil {
			return token, nil
		}
		// Don't return error - fall through to DB
	}
	fromDB = true
	loaded, loadErr, _ := tokenLoadGroup.Do(key, func() (interface{}, error) {
		var dbToken *Token
		err := DB.Where(commonKeyCol+" = ?", key).First(&dbToken).Error
		return dbToken, err
	})
	if loadErr != nil {
		return nil, loadErr
	}
	loadedToken, ok := loaded.(*Token)
	if !ok {
		return nil, fmt.Errorf("unexpected token cache load type %T", loaded)
	}
	return loadedToken, nil
}

func (token *Token) Insert() error {
	var err error
	err = DB.Create(token).Error
	return err
}

// Update Make sure your token's fields is completed, because this will update non-zero values
func (token *Token) Update() (err error) {
	defer func() {
		if shouldUpdateRedis(true, err) {
			gopool.Go(func() {
				err := cacheSetToken(*token)
				if err != nil {
					common.SysLog("failed to update token cache: " + err.Error())
				}
			})
		}
	}()
	err = DB.Model(token).Select("name", "status", "expired_time", "remain_quota", "unlimited_quota",
		"model_limits_enabled", "model_limits", "allow_ips", "group", "cross_group_retry",
		"quota_type", "window_hours", "window_quota", "window_start_hour",
		"cycle_days", "cycle_quota",
		"window_used_quota", "window_start_time", "cycle_used_quota", "cycle_start_time").Updates(token).Error
	return err
}

func (token *Token) SelectUpdate() (err error) {
	defer func() {
		if shouldUpdateRedis(true, err) {
			gopool.Go(func() {
				err := cacheSetToken(*token)
				if err != nil {
					common.SysLog("failed to update token cache: " + err.Error())
				}
			})
		}
	}()
	// This can update zero values
	return DB.Model(token).Select("accessed_time", "status").Updates(token).Error
}

func (token *Token) Delete() (err error) {
	defer func() {
		if shouldUpdateRedis(true, err) {
			gopool.Go(func() {
				err := cacheDeleteToken(token.Key)
				if err != nil {
					common.SysLog("failed to delete token cache: " + err.Error())
				}
			})
		}
	}()
	err = DB.Delete(token).Error
	return err
}

func (token *Token) IsModelLimitsEnabled() bool {
	return token.ModelLimitsEnabled
}

func (token *Token) GetModelLimits() []string {
	if token.ModelLimits == "" {
		return []string{}
	}
	return strings.Split(token.ModelLimits, ",")
}

func (token *Token) GetModelLimitsMap() map[string]bool {
	limits := token.GetModelLimits()
	limitsMap := make(map[string]bool)
	for _, limit := range limits {
		limitsMap[limit] = true
	}
	return limitsMap
}

func DisableModelLimits(tokenId int) error {
	token, err := GetTokenById(tokenId)
	if err != nil {
		return err
	}
	token.ModelLimitsEnabled = false
	token.ModelLimits = ""
	return token.Update()
}

func DeleteTokenById(id int, userId int) (err error) {
	// Why we need userId here? In case user want to delete other's token.
	if id == 0 || userId == 0 {
		return errors.New("id 或 userId 为空！")
	}
	token := Token{Id: id, UserId: userId}
	err = DB.Where(token).First(&token).Error
	if err != nil {
		return err
	}
	return token.Delete()
}

func IncreaseTokenQuota(id int, key string, quota int) (err error) {
	if quota < 0 {
		return errors.New("quota 不能为负数！")
	}
	if common.RedisEnabled {
		gopool.Go(func() {
			err := cacheIncrTokenQuota(key, int64(quota))
			if err != nil {
				common.SysLog("failed to increase token quota: " + err.Error())
			}
		})
	}
	if common.BatchUpdateEnabled {
		addNewRecord(BatchUpdateTypeTokenQuota, id, quota)
		return nil
	}
	return increaseTokenQuota(id, quota)
}

func increaseTokenQuota(id int, quota int) (err error) {
	err = DB.Model(&Token{}).Where("id = ?", id).Updates(
		map[string]interface{}{
			"remain_quota":  gorm.Expr("remain_quota + ?", quota),
			"used_quota":    gorm.Expr("used_quota - ?", quota),
			"accessed_time": common.GetTimestamp(),
		},
	).Error
	return err
}

func DecreaseTokenQuota(id int, key string, quota int) (err error) {
	if quota < 0 {
		return errors.New("quota 不能为负数！")
	}
	if common.RedisEnabled {
		gopool.Go(func() {
			err := cacheDecrTokenQuota(key, int64(quota))
			if err != nil {
				common.SysLog("failed to decrease token quota: " + err.Error())
			}
		})
	}
	if common.BatchUpdateEnabled {
		addNewRecord(BatchUpdateTypeTokenQuota, id, -quota)
		return nil
	}
	return decreaseTokenQuota(id, quota)
}

func decreaseTokenQuota(id int, quota int) (err error) {
	err = DB.Model(&Token{}).Where("id = ?", id).Updates(
		map[string]interface{}{
			"remain_quota":  gorm.Expr("remain_quota - ?", quota),
			"used_quota":    gorm.Expr("used_quota + ?", quota),
			"accessed_time": common.GetTimestamp(),
		},
	).Error
	return err
}

// IncreaseWindowQuota 增加窗口已用额度（退还额度时使用）
func IncreaseWindowQuota(id int, key string, quota int) (err error) {
	if quota < 0 {
		return errors.New("quota 不能为负数！")
	}
	if common.RedisEnabled {
		gopool.Go(func() {
			err := cacheIncrWindowUsedQuota(key, -int64(quota))
			if err != nil {
				common.SysLog("failed to increase window quota: " + err.Error())
			}
		})
	}
	if common.BatchUpdateEnabled {
		addNewRecord(BatchUpdateTypeWindowQuota, id, -quota)
		return nil
	}
	return increaseWindowQuota(id, -quota)
}

// DecreaseWindowQuota 减少窗口已用额度（扣费时使用）
func DecreaseWindowQuota(id int, key string, quota int) (err error) {
	if quota < 0 {
		return errors.New("quota 不能为负数！")
	}
	if common.RedisEnabled {
		ok, err := cacheDecrWindowQuotaCond(key, quota)
		if err != nil {
			if errors.Is(err, redis.Nil) {
				// 缓存未命中，回退到 DB/Batch 路径
				if common.BatchUpdateEnabled {
					addNewRecord(BatchUpdateTypeWindowQuota, id, quota)
					return nil
				}
				return decreaseWindowQuota(id, quota)
			}
			return err
		}
		if !ok {
			return errors.New("token window quota is not enough")
		}
		// Redis 扣减成功，异步同步到 DB
		if common.BatchUpdateEnabled {
			addNewRecord(BatchUpdateTypeWindowQuota, id, quota)
		} else {
			gopool.Go(func() {
				if err := increaseWindowQuota(id, quota); err != nil {
					common.SysLog("failed to sync window quota to db: " + err.Error())
				}
			})
		}
		return nil
	}
	if common.BatchUpdateEnabled {
		addNewRecord(BatchUpdateTypeWindowQuota, id, quota)
		return nil
	}
	return decreaseWindowQuota(id, quota)
}

func decreaseWindowQuota(id int, quota int) (err error) {
	result := DB.Model(&Token{}).Where("id = ? AND window_used_quota + ? <= window_quota", id, quota).Updates(
		map[string]interface{}{
			"window_used_quota": gorm.Expr("window_used_quota + ?", quota),
			"accessed_time":     common.GetTimestamp(),
		},
	)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("token window quota is not enough")
	}
	return nil
}

func increaseWindowQuota(id int, quota int) (err error) {
	updates := map[string]interface{}{
		"window_used_quota": gorm.Expr("window_used_quota + ?", quota),
		"accessed_time":     common.GetTimestamp(),
	}
	err = DB.Model(&Token{}).Where("id = ?", id).Updates(updates).Error
	return err
}

// IncreaseCycleQuota 增加周期已用额度（退还额度时使用，传入负值）
func IncreaseCycleQuota(id int, key string, quota int) (err error) {
	if quota < 0 {
		return errors.New("quota 不能为负数！")
	}
	if common.RedisEnabled {
		gopool.Go(func() {
			err := cacheIncrCycleUsedQuota(key, -int64(quota))
			if err != nil {
				common.SysLog("failed to increase cycle quota: " + err.Error())
			}
		})
	}
	if common.BatchUpdateEnabled {
		addNewRecord(BatchUpdateTypeCycleQuota, id, -quota)
		return nil
	}
	return increaseCycleQuota(id, -quota)
}

// DecreaseCycleQuota 减少周期已用额度（扣费时使用）
func DecreaseCycleQuota(id int, key string, quota int) (err error) {
	if quota < 0 {
		return errors.New("quota 不能为负数！")
	}
	if common.RedisEnabled {
		ok, err := cacheDecrCycleQuotaCond(key, quota)
		if err != nil {
			if errors.Is(err, redis.Nil) {
				if common.BatchUpdateEnabled {
					addNewRecord(BatchUpdateTypeCycleQuota, id, quota)
					return nil
				}
				return decreaseCycleQuota(id, quota)
			}
			return err
		}
		if !ok {
			return errors.New("token cycle quota is not enough")
		}
		if common.BatchUpdateEnabled {
			addNewRecord(BatchUpdateTypeCycleQuota, id, quota)
		} else {
			gopool.Go(func() {
				if err := increaseCycleQuota(id, quota); err != nil {
					common.SysLog("failed to sync cycle quota to db: " + err.Error())
				}
			})
		}
		return nil
	}
	if common.BatchUpdateEnabled {
		addNewRecord(BatchUpdateTypeCycleQuota, id, quota)
		return nil
	}
	return decreaseCycleQuota(id, quota)
}

func decreaseCycleQuota(id int, quota int) (err error) {
	result := DB.Model(&Token{}).Where("id = ? AND cycle_used_quota + ? <= cycle_quota", id, quota).Updates(
		map[string]interface{}{
			"cycle_used_quota": gorm.Expr("cycle_used_quota + ?", quota),
			"accessed_time":    common.GetTimestamp(),
		},
	)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("token cycle quota is not enough")
	}
	return nil
}

func increaseCycleQuota(id int, quota int) (err error) {
	err = DB.Model(&Token{}).Where("id = ?", id).Updates(
		map[string]interface{}{
			"cycle_used_quota": gorm.Expr("cycle_used_quota + ?", quota),
			"accessed_time":    common.GetTimestamp(),
		},
	).Error
	return err
}

// ResetWindowQuota 重置窗口额度到新窗口，仅在旧的 window_start_time 匹配时才执行，防止并发边界覆盖其他请求已扣减的额度。
func ResetWindowQuota(id int, oldStart int64, newStart int64) (err error) {
	result := DB.Model(&Token{}).Where("id = ? AND window_start_time = ?", id, oldStart).Updates(
		map[string]interface{}{
			"window_used_quota": 0,
			"window_start_time": newStart,
		},
	)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("window already reset by another request")
	}
	return nil
}

// ResetCycleQuota 重置周期额度到新周期，仅在旧的 cycle_start_time 匹配时才执行，防止并发边界覆盖其他请求已扣减的额度。
func ResetCycleQuota(id int, oldStart int64, newStart int64) (err error) {
	result := DB.Model(&Token{}).Where("id = ? AND cycle_start_time = ?", id, oldStart).Updates(
		map[string]interface{}{
			"cycle_used_quota": 0,
			"cycle_start_time": newStart,
		},
	)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("cycle already reset by another request")
	}
	return nil
}

// CountUserTokens returns total number of tokens for the given user, used for pagination
func CountUserTokens(userId int) (int64, error) {
	var total int64
	err := DB.Model(&Token{}).Where("user_id = ?", userId).Count(&total).Error
	return total, err
}

// ResetTokenKey 重置令牌密钥，仅更新 key 字段，其他字段不变
// 需要 userId 参数以验证令牌归属，防止越权操作
func ResetTokenKey(id int, userId int) (newKey string, err error) {
	if id == 0 || userId == 0 {
		return "", errors.New("id 或 userId 为空！")
	}

	// 按 token ID 加锁，串行化同一令牌的并发重置请求
	mu := getTokenResetLock(id)
	mu.Lock()
	defer mu.Unlock()

	// 先通过 id+userId 查询，验证令牌归属
	token := Token{Id: id, UserId: userId}
	err = DB.First(&token, "id = ? and user_id = ?", id, userId).Error
	if err != nil {
		return "", err
	}
	// 生成新 key
	newKey, err = common.GenerateKey()
	if err != nil {
		return "", err
	}
	oldKey := token.Key
	// 更新数据库中的 key
	err = DB.Model(&token).Update("key", newKey).Error
	if err != nil {
		return "", err
	}
	token.Key = newKey

	// 在锁内同步完成旧缓存删除 + 新缓存写入，确保任意时刻只有一个有效 key
	if common.RedisEnabled {
		if delErr := cacheDeleteToken(oldKey); delErr != nil {
			common.SysError("failed to delete old token cache after reset key: " + delErr.Error())
		}
		if setErr := cacheSetToken(token); setErr != nil {
			common.SysError("failed to update token cache after reset key: " + setErr.Error())
		}
	}
	return newKey, nil
}

// BatchDeleteTokens 删除指定用户的一组令牌，返回成功删除数量
func BatchDeleteTokens(ids []int, userId int) (int, error) {
	if len(ids) == 0 {
		return 0, errors.New("ids 不能为空！")
	}

	tx := DB.Begin()

	var tokens []Token
	if err := tx.Where("user_id = ? AND id IN (?)", userId, ids).Find(&tokens).Error; err != nil {
		tx.Rollback()
		return 0, err
	}

	if err := tx.Where("user_id = ? AND id IN (?)", userId, ids).Delete(&Token{}).Error; err != nil {
		tx.Rollback()
		return 0, err
	}

	if err := tx.Commit().Error; err != nil {
		return 0, err
	}

	if common.RedisEnabled {
		gopool.Go(func() {
			for _, t := range tokens {
				_ = cacheDeleteToken(t.Key)
			}
		})
	}

	return len(tokens), nil
}
