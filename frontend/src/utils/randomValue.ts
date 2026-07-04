import type { ProtoField } from '@/types/proto'
import { wktScalarCfg } from './wktScalar'
import { formatRFC3339WithLocalOffset } from './dateFormat'

export function randomValue(field: ProtoField): string {
  const t = field.type
  const wkt = wktScalarCfg(field)
  if (wkt) {
    switch (wkt.kind) {
      case 'timestamp':
        return formatRFC3339WithLocalOffset(new Date())
      case 'string':
        if (field.messageType === 'google.protobuf.Duration') {
          return `${1 + Math.floor(Math.random() * 100)}s`
        }
        return 'value'
      case 'bool':
        return Math.random() < 0.5 ? 'true' : 'false'
      case 'int32':
        return String(Math.floor(Math.random() * 2001) - 1000)
      case 'int64':
        return String(Math.floor(Math.random() * 200001) - 100000)
      case 'uint32':
        return String(Math.floor(Math.random() * 2001))
      case 'uint64':
        return String(Math.floor(Math.random() * 200001))
      case 'float':
        return (Math.random() * 200 - 100).toFixed(2)
      case 'double':
        return (Math.random() * 200000 - 100000).toFixed(4)
      case 'bytes': {
        const len = 4 + Math.floor(Math.random() * 8)
        const bytes = new Uint8Array(len)
        crypto.getRandomValues(bytes)
        return btoa(String.fromCharCode(...bytes))
      }
      case 'jsonObject':
        if (field.messageType === 'google.protobuf.Any') return '{}'
        return '{"key":"value"}'
      case 'jsonArray':
        return '[1,true,"x"]'
      case 'jsonValue':
        return '"value"'
      default:
        return ''
    }
  }
  if (t === 'bool') return Math.random() < 0.5 ? 'true' : 'false'
  if (t === 'enum' && field.enumValues?.length) {
    const ev = field.enumValues[Math.floor(Math.random() * field.enumValues.length)]
    return ev.name
  }
  if (t === 'string') {
    const chars = 'abcdefghijklmnopqrstuvwxyz'
    const len = 4 + Math.floor(Math.random() * 8)
    let s = ''
    for (let i = 0; i < len; i++) s += chars[Math.floor(Math.random() * chars.length)]
    return s
  }
  if (t === 'bytes') {
    const len = 4 + Math.floor(Math.random() * 8)
    const bytes = new Uint8Array(len)
    crypto.getRandomValues(bytes)
    return btoa(String.fromCharCode(...bytes))
  }
  if (['int32', 'sint32', 'sfixed32'].includes(t))
    return String(Math.floor(Math.random() * 2001) - 1000)
  if (['int64', 'sint64', 'sfixed64'].includes(t))
    return String(Math.floor(Math.random() * 200001) - 100000)
  if (['uint32', 'fixed32'].includes(t)) return String(Math.floor(Math.random() * 2001))
  if (['uint64', 'fixed64'].includes(t)) return String(Math.floor(Math.random() * 200001))
  if (t === 'float') return (Math.random() * 200 - 100).toFixed(2)
  if (t === 'double') return (Math.random() * 200000 - 100000).toFixed(4)
  return ''
}
