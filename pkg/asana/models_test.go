package asana

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPaginationDataIsPresent(t *testing.T) {
	for _, tc := range []struct {
		name    string
		body    string
		present bool
		offset  string
	}{
		{name: "object", body: `{"next_page":{"offset":"abc"}}`, present: true, offset: "abc"},
		{name: "empty object", body: `{"next_page":{}}`, present: true},
		{name: "explicit null", body: `{"next_page":null}`, present: true},
		{name: "key absent", body: `{}`, present: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var res UsersResponse
			require.NoError(t, json.Unmarshal([]byte(tc.body), &res))
			require.Equal(t, tc.present, res.NextPage.IsPresent())
			require.Equal(t, tc.present, res.HasPaginationData())
			require.Equal(t, tc.offset, res.NextPage.Offset)
		})
	}
}
