package serializer

import (
	"encoding/binary"
	"fmt"
)

type BitboardSerializer struct{}

func (b BitboardSerializer) Serialize(value interface{}) ([]byte, error) {
	if v, ok := value.(uint64); ok {
		data := make([]byte, 8)
		binary.BigEndian.PutUint64(data, v)
		return data, nil
	}
	return nil, fmt.Errorf("failed to serialize value: %v", value)
}

func (b BitboardSerializer) Deserialize(data []byte, dest interface{}) error {
	if v, ok := dest.(*uint64); ok && len(data) == 8 {
		*v = binary.BigEndian.Uint64(data)
		return nil
	}
	return fmt.Errorf("failed to deserialize data: %v", data)
}
