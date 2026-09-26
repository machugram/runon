package inventory

import (
	"strings"
	"testing"
)

func TestParseAndRender(t *testing.T) {
	raw := []byte(`{
	  "inventory": {"value": {
	    "proxy": [{"name":"dev-proxy","host":"127.0.0.1","port":2201,"user":"root"}],
	    "app": [{"name":"dev-app-1","host":"127.0.0.1","port":2202,"user":"root"}]
	  }},
	  "ansible": {"sensitive": true, "value": {
	    "env_name": "dev",
	    "db_host": "dev-postgres",
	    "db_port": 5432,
	    "db_admin_user": "platform",
	    "db_admin_password": "secret",
	    "db_name": "platform",
	    "db_user": "platform",
	    "db_password": "secret",
	    "ssh_private_key_path": "/tmp/lab/id_ed25519",
	    "proxy_http_port": 18080
	  }}
	}`)
	inv, vars, err := ParseTerraformOutput(raw)
	if err != nil {
		t.Fatal(err)
	}
	ini, err := RenderINI(inv, vars.SSHPrivateKeyPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(ini, "dev-app-1 ansible_host=127.0.0.1 ansible_port=2202") {
		t.Fatalf("inventory:\n%s", ini)
	}
	if !strings.Contains(ini, "[proxy]\ndev-proxy") {
		t.Fatalf("proxy group:\n%s", ini)
	}
	vars.ServerBinary = "/work/dist/server"
	extra, err := RenderExtraVars(vars)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(extra), `"server_binary": "/work/dist/server"`) {
		t.Fatalf("extra vars: %s", extra)
	}
	if vars.ProxyHTTPPort != 18080 {
		t.Fatalf("port %d", vars.ProxyHTTPPort)
	}
}

func TestRenderINIRejectsSpacesInKeyPath(t *testing.T) {
	inv := Inventory{Proxy: []Host{{Name: "p", Host: "127.0.0.1", Port: 22}}, App: []Host{{Name: "a", Host: "127.0.0.1", Port: 22}}}
	if _, err := RenderINI(inv, "/tmp/my key"); err == nil {
		t.Fatal("expected key path with a space to fail")
	}
}
