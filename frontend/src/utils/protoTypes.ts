export const INT_TYPES = [
  'int32',
  'int64',
  'sint32',
  'sint64',
  'sfixed32',
  'sfixed64',
  'fixed32',
  'fixed64',
]
export const UINT_TYPES = ['uint32', 'uint64']
export const FLOAT_TYPES = ['float', 'double']
export const ALL_NUMERIC = [...INT_TYPES, ...UINT_TYPES, ...FLOAT_TYPES]

export function isNumericProtoType(type: string): boolean {
  return ALL_NUMERIC.includes(type)
}
