package inventory

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

type Host struct {
	Name string `json:"name"`
	Host string `json:"host"`
	Port int    `json:"port"`
	User string `json:"user"`
}

type Inventory struct {
	Proxy []Host `json:"proxy"`
	App   []Host `json:"app"`
}

// Vars is the Ansible extra-vars file. ServerBinary and AppListenPort are filled by the CLI.
type Vars struct {
	EnvName           string `json:"env_name"`
	DBHost            string `json:"db_host"`
	DBPort            int    `json:"db_port"`
	DBAdminUser       string `json:"db_admin_user"`
	DBAdminPassword   string `json:"db_admin_password"`
	DBName            string `json:"db_name"`
	DBUser            string `json:"db_user"`
	DBPassword        string `json:"db_password"`
	SSHPrivateKeyPath string `json:"ssh_private_key_path"`
	ProxyHTTPPort     int    `json:"proxy_http_port"`
	ServerBinary      string `json:"server_binary"`
	AppListenPort     int    `json:"app_listen_port"`
}

type tfValue[T any] struct {
	Value T `json:"value"`
}

type tfOutput struct {
	Inventory tfValue[Inventory] `json:"inventory"`
	Ansible   tfValue[Vars]      `json:"ansible"`
}

func ParseTerraformOutput(data []byte) (Inventory, Vars, error) {
	var out tfOutput
	if err := json.Unmarshal(data, &out); err != nil {
		return Inventory{}, Vars{}, fmt.Errorf("parse terraform output: %w", err)
	}
	if len(out.Inventory.Value.App) == 0 || len(out.Inventory.Value.Proxy) == 0 {
		return Inventory{}, Vars{}, fmt.Errorf("terraform inventory is missing app or proxy hosts")
	}
	if out.Ansible.Value.SSHPrivateKeyPath == "" || out.Ansible.Value.DBPassword == "" {
		return Inventory{}, Vars{}, fmt.Errorf("terraform ansible output is missing secrets")
	}
	if out.Ansible.Value.ProxyHTTPPort == 0 {
		return Inventory{}, Vars{}, fmt.Errorf("terraform ansible output is missing proxy_http_port")
	}
	return out.Inventory.Value, out.Ansible.Value, nil
}

func RenderINI(inv Inventory, privateKeyPath string) (string, error) {
	if privateKeyPath == "" {
		return "", fmt.Errorf("ssh private key path is empty")
	}
	if strings.ContainsAny(privateKeyPath, " \t'\"") {
		return "", fmt.Errorf("ssh private key path must not contain spaces or quotes")
	}
	var b strings.Builder
	b.WriteString("[all:vars]\n")
	b.WriteString("ansible_user=root\n")
	fmt.Fprintf(&b, "ansible_ssh_private_key_file=%s\n", privateKeyPath)
	b.WriteString("ansible_ssh_common_args='-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o IdentitiesOnly=yes'\n")
	b.WriteString("ansible_python_interpreter=/usr/bin/python3\n\n")
	if err := writeGroup(&b, "proxy", inv.Proxy); err != nil {
		return "", err
	}
	b.WriteByte('\n')
	if err := writeGroup(&b, "app", inv.App); err != nil {
		return "", err
	}
	return b.String(), nil
}

func writeGroup(b *strings.Builder, name string, hosts []Host) error {
	if len(hosts) == 0 {
		return fmt.Errorf("inventory group %s is empty", name)
	}
	fmt.Fprintf(b, "[%s]\n", name)
	for _, host := range hosts {
		if host.Name == "" || host.Host == "" || host.Port == 0 {
			return fmt.Errorf("inventory host in %s is incomplete", name)
		}
		user := host.User
		if user == "" {
			user = "root"
		}
		fmt.Fprintf(b, "%s ansible_host=%s ansible_port=%d ansible_user=%s\n", host.Name, host.Host, host.Port, user)
	}
	return nil
}

func RenderExtraVars(v Vars) ([]byte, error) {
	if v.ServerBinary == "" {
		return nil, fmt.Errorf("server binary path is empty")
	}
	if v.AppListenPort == 0 {
		v.AppListenPort = 8080
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
