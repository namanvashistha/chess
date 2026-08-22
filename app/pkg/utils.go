package pkg

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"strings"

	petname "github.com/dustinkirkland/golang-petname"
)

// randomStringFrom returns a cryptographically random string of the given
// length drawn from charset. Session tokens and invite codes are guessable
// credentials, so they must not come from math/rand: that generator is
// deterministic and was previously seeded from time.Now(), which made a token
// recoverable from its creation time (and made two tokens minted in the same
// clock tick identical).
func randomStringFrom(charset string, length int) string {
	result := make([]byte, length)
	max := big.NewInt(int64(len(charset)))
	for i := range result {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			// crypto/rand fails only if the OS entropy source is unavailable,
			// in which case we must not fall back to a weak generator.
			panic("pkg: crypto/rand unavailable: " + err.Error())
		}
		result[i] = charset[n.Int64()]
	}
	return string(result)
}

func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	return randomStringFrom(charset, length)
}

func GenerateRandomNumericString(length int) string {
	return randomStringFrom("0123456789", length)
}

func GenerateRandomBool() bool {
	n, err := rand.Int(rand.Reader, big.NewInt(2))
	if err != nil {
		panic("pkg: crypto/rand unavailable: " + err.Error())
	}
	return n.Int64() == 0
}

func GenerateRandomUserName() string {
	return fmt.Sprintf("%s-%s", petname.Generate(2, "-"), GenerateRandomNumericString(4))
}

// BindPayloadToStruct copies string values out of a decoded JSON object into the
// matching json-tagged string fields of obj.
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

		// The json tag may carry options (e.g. "promotion,omitempty"); the key
		// is just the part before the first comma. "-" means "skip this field".
		fieldName := field.Tag.Get("json")
		if comma := strings.IndexByte(fieldName, ','); comma >= 0 {
			fieldName = fieldName[:comma]
		}
		if fieldName == "" || fieldName == "-" {
			continue
		}

		// Optional fields: a key absent from the payload is left as its zero
		// value rather than failing the whole bind. This keeps adding new
		// optional fields (e.g. "promotion") from breaking existing clients.
		raw, exists := payload[fieldName]
		if !exists {
			continue
		}
		strValue, ok := raw.(string)
		if !ok {
			return fmt.Errorf("failed to bind key '%s': not a string, found %T", fieldName, raw)
		}

		// Only string fields can be bound; anything else would panic on Set.
		target := val.Elem().Field(i)
		if target.Kind() != reflect.String {
			return fmt.Errorf("failed to bind key '%s': target field is %s, not a string", fieldName, target.Kind())
		}
		target.SetString(strValue)
	}

	return nil
}
