package model

import (
	"errors"
	"fmt"
	"math/rand"
	"sort"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

type RandomSatisfiedChannelByAPITypeParams struct {
	Group     string
	ModelName string
	APIType   int
	Retry     int
}

func GetRandomSatisfiedChannelByAPIType(p RandomSatisfiedChannelByAPITypeParams) (*Channel, error) {
	if !common.MemoryCacheEnabled {
		return getChannelByAPITypeDB(p)
	}

	channelSyncLock.RLock()
	defer channelSyncLock.RUnlock()

	matchedModel := p.ModelName
	ids := group2model2channels[p.Group][matchedModel]
	if len(ids) == 0 {
		normalized := ratio_setting.FormatMatchingModelName(p.ModelName)
		ids = group2model2channels[p.Group][normalized]
		if len(ids) > 0 {
			matchedModel = normalized
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}

	candidates, err := filterChannelsByAPIType(ids, p.APIType)
	if err != nil {
		return nil, err
	}
	return chooseRandomChannelByPriorityAndWeight(channelPickContext{
		group:     p.Group,
		modelName: matchedModel,
		retry:     p.Retry,
	}, candidates)
}

func filterChannelsByAPIType(channelIDs []int, apiType int) ([]*Channel, error) {
	out := make([]*Channel, 0, len(channelIDs))
	for _, id := range channelIDs {
		ch, ok := channelsIDM[id]
		if !ok {
			return nil, fmt.Errorf("database inconsistency: channel #%d not found", id)
		}
		chApiType, ok := common.ChannelType2APIType(ch.Type)
		if !ok || chApiType != apiType {
			continue
		}
		out = append(out, ch)
	}
	return out, nil
}

type channelPickContext struct {
	group     string
	modelName string
	retry     int
}

func chooseRandomChannelByPriorityAndWeight(ctx channelPickContext, channels []*Channel) (*Channel, error) {
	if len(channels) == 0 {
		return nil, nil
	}
	if len(channels) == 1 {
		return channels[0], nil
	}

	priorities := uniqueSortedPriorities(channels)
	if len(priorities) == 0 {
		return nil, nil
	}
	retryIndex := ctx.retry
	if retryIndex >= len(priorities) {
		retryIndex = len(priorities) - 1
	}
	targetPriority := int64(priorities[retryIndex])

	target, sumWeight := pickChannelsByPriority(channels, targetPriority)
	if len(target) == 0 {
		return nil, fmt.Errorf("no channel found, group: %s, model: %s, priority: %d", ctx.group, ctx.modelName, targetPriority)
	}

	return weightedPickChannel(target, sumWeight)
}

func uniqueSortedPriorities(channels []*Channel) []int {
	unique := make(map[int]struct{}, len(channels))
	for _, ch := range channels {
		unique[int(ch.GetPriority())] = struct{}{}
	}
	out := make([]int, 0, len(unique))
	for p := range unique {
		out = append(out, p)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(out)))
	return out
}

func pickChannelsByPriority(channels []*Channel, priority int64) ([]*Channel, int) {
	out := make([]*Channel, 0, len(channels))
	sumWeight := 0
	for _, ch := range channels {
		if ch.GetPriority() != priority {
			continue
		}
		sumWeight += ch.GetWeight()
		out = append(out, ch)
	}
	return out, sumWeight
}

func weightedPickChannel(channels []*Channel, sumWeight int) (*Channel, error) {
	if len(channels) == 0 {
		return nil, errors.New("channel not found")
	}

	smoothingFactor := 1
	smoothingAdjustment := 0
	if sumWeight == 0 {
		sumWeight = len(channels) * 100
		smoothingAdjustment = 100
	} else if sumWeight/len(channels) < 10 {
		smoothingFactor = 100
	}

	totalWeight := sumWeight * smoothingFactor
	randomWeight := rand.Intn(totalWeight)
	for _, ch := range channels {
		randomWeight -= ch.GetWeight()*smoothingFactor + smoothingAdjustment
		if randomWeight < 0 {
			return ch, nil
		}
	}
	return nil, errors.New("channel not found")
}

func channelTypesForAPIType(apiType int) []int {
	out := make([]int, 0)
	for channelType := 1; channelType < constant.ChannelTypeDummy; channelType++ {
		mapped, ok := common.ChannelType2APIType(channelType)
		if ok && mapped == apiType {
			out = append(out, channelType)
		}
	}
	return out
}

func getChannelByAPITypeDB(p RandomSatisfiedChannelByAPITypeParams) (*Channel, error) {
	allowedTypes := channelTypesForAPIType(p.APIType)
	if len(allowedTypes) == 0 {
		return nil, fmt.Errorf("unsupported apiType: %d", p.APIType)
	}

	query := channelByAPITypeDBQuery{
		group:        p.Group,
		modelName:    p.ModelName,
		allowedTypes: allowedTypes,
		retry:        p.Retry,
	}
	channel, err := getChannelByAPITypeDBWithModelName(query)
	if err != nil || channel != nil {
		return channel, err
	}
	normalized := ratio_setting.FormatMatchingModelName(p.ModelName)
	if normalized == "" || normalized == p.ModelName {
		return nil, nil
	}
	query.modelName = normalized
	return getChannelByAPITypeDBWithModelName(query)
}

type channelByAPITypeDBQuery struct {
	group        string
	modelName    string
	allowedTypes []int
	retry        int
}

func getChannelByAPITypeDBWithModelName(q channelByAPITypeDBQuery) (*Channel, error) {
	priorities, err := listPrioritiesByAPIType(q.group, q.modelName, q.allowedTypes)
	if err != nil {
		return nil, err
	}
	if len(priorities) == 0 {
		return nil, nil
	}
	retryIndex := q.retry
	if retryIndex >= len(priorities) {
		retryIndex = len(priorities) - 1
	}
	priorityToUse := priorities[retryIndex]

	var abilities []Ability
	err = DB.Model(&Ability{}).
		Select("abilities.*").
		Joins("JOIN channels ON channels.id = abilities.channel_id").
		Where("abilities."+commonGroupCol+" = ? AND abilities.model = ? AND abilities.enabled = ? AND abilities.priority = ? AND channels.status = ? AND channels.type IN (?)",
			q.group, q.modelName, true, priorityToUse, common.ChannelStatusEnabled, q.allowedTypes,
		).
		Order("weight DESC").
		Find(&abilities).Error
	if err != nil {
		return nil, err
	}
	if len(abilities) == 0 {
		return nil, nil
	}

	channelID := chooseChannelIDByAbilityWeight(abilities)
	if channelID == 0 {
		return nil, nil
	}

	var ch Channel
	err = DB.First(&ch, "id = ?", channelID).Error
	if err != nil {
		return nil, err
	}
	return &ch, nil
}

func listPrioritiesByAPIType(group string, modelName string, allowedTypes []int) ([]int, error) {
	var priorities []int
	err := DB.Model(&Ability{}).
		Select("DISTINCT(abilities.priority)").
		Joins("JOIN channels ON channels.id = abilities.channel_id").
		Where("abilities."+commonGroupCol+" = ? AND abilities.model = ? AND abilities.enabled = ? AND channels.status = ? AND channels.type IN (?)",
			group, modelName, true, common.ChannelStatusEnabled, allowedTypes,
		).
		Order("abilities.priority DESC").
		Pluck("abilities.priority", &priorities).Error
	return priorities, err
}

func chooseChannelIDByAbilityWeight(abilities []Ability) int {
	weightSum := uint(0)
	for _, a := range abilities {
		weightSum += a.Weight + 10
	}
	if weightSum == 0 {
		return 0
	}

	weight := common.GetRandomInt(int(weightSum))
	for _, a := range abilities {
		weight -= int(a.Weight) + 10
		if weight <= 0 {
			return a.ChannelId
		}
	}
	return 0
}
