package pkg

import (
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"strconv"
	"time"

	petname "github.com/dustinkirkland/golang-petname"
)

func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	seed := rand.NewSource(time.Now().UnixNano())
	random := rand.New(seed)

	result := make([]byte, length)
	for i := range result {
		result[i] = charset[random.Intn(len(charset))]
	}
	return string(result)
}

func GenerateRandomNumericString(length int) string {
	const charset = "0123456789"
	seed := rand.NewSource(time.Now().UnixNano())
	random := rand.New(seed)

	result := make([]byte, length)
	for i := range result {
		result[i] = charset[random.Intn(len(charset))]
	}
	return string(result)
}

func GenerateRandomBool() bool {
	seed := rand.NewSource(time.Now().UnixNano())
	random := rand.New(seed)
	return random.Intn(2) == 0
}

func GenerateRandomUserName() string {
	return fmt.Sprintf("%s-%s", petname.Generate(2, "-"), GenerateRandomNumericString(4))
}

func ExtractString(payload map[string]interface{}, key string) (string, error) {
	value, exists := payload[key]
	if !exists {
		return "", fmt.Errorf("missing key: %s", key)
	}

	strValue, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("key '%s' is not a string, found type %T", key, value)
	}

	return strValue, nil
}

// Generic function to bind the payload to any struct
func BindPayloadToStruct(payload map[string]interface{}, obj interface{}) error {
	// Ensure the object is a pointer to a struct
	val := reflect.ValueOf(obj)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return errors.New("provided object is not a pointer to a struct")
	}

	// Get the struct type and iterate through its fields
	typ := val.Elem().Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldName := field.Tag.Get("json")

		// Extract value from payload
		fieldValue, err := ExtractString(payload, fieldName)
		if err != nil {
			return fmt.Errorf("failed to bind key '%s': %v", fieldName, err)
		}

		// Set the struct field
		val.Elem().Field(i).SetString(fieldValue)
	}

	return nil
}

func ConvertUint64ToString(input interface{}) {
	// Get the value of the input
	val := reflect.ValueOf(input)

	// Ensure that input is a pointer to a struct
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		fmt.Println("Input must be a pointer to a struct.")
		return
	}

	// Get the value of the struct (dereferencing the pointer)
	val = val.Elem()

	// Iterate over all fields of the struct
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)

		// If the field is of type uint64, convert it to string
		if field.Kind() == reflect.Uint64 {
			strValue := strconv.FormatUint(field.Uint(), 10)
			// Create a string field to store the converted value
			stringField := reflect.ValueOf(&strValue).Elem()
			// Set the field with the string value
			field.Set(stringField)
		}
	}
}
