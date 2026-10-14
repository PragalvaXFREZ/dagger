package cloud

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSourcesProviderMetadataCompatibility(t *testing.T) {
	for _, tc := range []struct {
		name, apiError string
		fallback       bool
	}{
		{name: "old schema", apiError: `Cannot query field "provider" on type "Source".`, fallback: true},
		{name: "authorization failure", apiError: "unauthorized"},
		{name: "unrelated schema failure", apiError: `Cannot query field "sources" on type "Query".`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var queries []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var request struct{ Query string }
				require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
				queries = append(queries, request.Query)
				w.Header().Set("Content-Type", "application/json")
				if len(queries) == 1 {
					fmt.Fprintf(w, `{"errors":[{"message":%q}]}`, tc.apiError)
					return
				}
				fmt.Fprint(w, `{"data":{"sources":[{"id":"123","name":"acme","type":"Organization"}]}}`)
			}))
			t.Cleanup(server.Close)
			sources, err := testClient(t, server.URL).Sources(t.Context())
			if tc.fallback {
				require.NoError(t, err)
				require.Len(t, queries, 2)
				require.NotContains(t, queries[1], "\t\tprovider\n")
				require.NotContains(t, queries[1], "\t\tinstance\n")
				require.Len(t, sources, 1)
				require.Equal(t, "123", sources[0].ID)
			} else {
				require.Error(t, err)
				require.True(t, strings.Contains(err.Error(), tc.apiError))
				require.Len(t, queries, 1)
			}
		})
	}
}
