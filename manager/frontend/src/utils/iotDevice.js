export const getDeviceNickName = (device) => String(device?.nick_name || '').trim()

export const getDeviceSN = (device) => String(device?.device_name || '').trim()

export const formatDeviceNickName = (device, emptyLabel = '未设置') => {
  const nickName = getDeviceNickName(device)
  return nickName || emptyLabel
}

export const getDeviceReferenceLabel = (device) => {
  const nickName = getDeviceNickName(device)
  if (nickName) return nickName
  const sn = getDeviceSN(device)
  return sn || '未命名设备'
}

export const formatDeviceOptionLabel = (device, emptyNickLabel = '未设置') => {
  const nickName = formatDeviceNickName(device, emptyNickLabel)
  const sn = getDeviceSN(device) || '-'
  return `${nickName} (${sn})`
}
