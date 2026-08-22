package pkg

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// PanicHandler used to do strings.Split(msg, ":")[1] unconditionally. Any panic
// value without a colon made the handler itself panic *inside* recover(), which
// crashes the process instead of returning a 500.
func TestPanicHandlerSurvivesArbitraryPanicValues(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cases := []struct {
		name     string
		panicVal interface{}
		wantCode int
	}{
		{"no colon", "boom", http.StatusInternalServerError},
		{"empty string", "", http.StatusInternalServerError},
		{"non-string", 42, http.StatusInternalServerError},
		{"nil map write", map[string]string(nil), http.StatusInternalServerError},
		{"known key", "DATA_NOT_FOUND: nope", http.StatusBadRequest},
		{"unauthorized", "UNAUTHORIZED: nope", http.StatusUnauthorized},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/x", func(ctx *gin.Context) {
				defer PanicHandler(ctx)
				panic(c.panicVal)
			})

			w := httptest.NewRecorder()
			// If PanicHandler re-panics, this call does not return normally.
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))

			if w.Code != c.wantCode {
				t.Errorf("status = %d, want %d (body %q)", w.Code, c.wantCode, w.Body.String())
			}
		})
	}
}
