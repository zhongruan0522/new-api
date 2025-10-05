const { contextBridge } = require('electron');

// 获取数据目录路径（用于显示给用户）
// 使用字符串拼接而不是 path.join 避免模块依赖问题
function getDataDirPath() {
  const platform = process.platform;
  const homeDir = process.env.HOME || process.env.USERPROFILE || '';
  
  switch (platform) {
    case 'darwin':
      return `${homeDir}/Library/Application Support/New API/data`;
    case 'win32': {
      const appData = process.env.APPDATA || `${homeDir}\\AppData\\Roaming`;
      return `${appData}\\New API\\data`;
    }
    case 'linux':
      return `${homeDir}/.config/New API/data`;
    default:
      return `${homeDir}/.new-api/data`;
  }
}

contextBridge.exposeInMainWorld('electron', {
  isElectron: true,
  version: process.versions.electron,
  platform: process.platform,
  versions: process.versions,
  dataDir: getDataDirPath()
});