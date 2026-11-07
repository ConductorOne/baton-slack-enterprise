package enterprise

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	fakeBot  = "xoxb-bot"
	fakeUser = "xoxp-user"
)

func TestGetAuthTeamsList_UsesBotToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, UrlPathAuthTeamsList, r.URL.Path)
		assert.NoError(t, r.ParseForm())
		assert.Equal(t, "next-page", r.PostForm.Get("cursor"))
		assert.Equal(t, "Bearer "+fakeBot, r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"teams":[{"id":"T1","name":"Member Workspace"},{"id":"T2","name":"Other Workspace"}],"response_metadata":{"next_cursor":"cursor-2"}}`))
	}))
	defer server.Close()

	c, err := NewClient(server.Client(), fakeUser, fakeBot, "E1", false, server.URL)
	require.NoError(t, err)

	teams, nextCursor, _, err := c.GetAuthTeamsList(context.Background(), "next-page")
	require.NoError(t, err)
	require.Len(t, teams, 2)
	require.Equal(t, "T2", teams[1].ID)
	require.Equal(t, "Other Workspace", teams[1].Name)
	require.Equal(t, "cursor-2", nextCursor)
}
