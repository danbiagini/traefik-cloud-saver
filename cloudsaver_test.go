package traefik_cloud_saver

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/traefik/genconf/dynamic"
)

func TestNew(t *testing.T) {
	config := CreateConfig()
	config.PollInterval = "1s"

	provider, err := New(context.Background(), config, "test")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		err = provider.Stop()
		if err != nil {
			t.Fatal(err)
		}
	})

	err = provider.Init()
	if err != nil {
		t.Fatal(err)
	}

	cfgChan := make(chan json.Marshaler)

	err = provider.Provide(cfgChan)
	if err != nil {
		t.Fatal(err)
	}

	data := <-cfgChan

	expected := &dynamic.Configuration{
        HTTP: &dynamic.HTTPConfiguration{
            Routers:           make(map[string]*dynamic.Router),
            Services:          make(map[string]*dynamic.Service),
            Middlewares:       make(map[string]*dynamic.Middleware),
            ServersTransports: make(map[string]*dynamic.ServersTransport),
        },
	}

	expectedJSON, err := json.MarshalIndent(expected, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	dataJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(expectedJSON, dataJSON) {
		t.Fatalf("got %s, want: %s", string(dataJSON), string(expectedJSON))
	}
}
