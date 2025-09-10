package main

import (
	"fmt"
	"strings"
)

func main() {
	playbook := `
- name: Set network device configuration
  hosts: all
  become: true
  vars:
    mac_types: {}
  tasks:
    - name: Set network configuration for unconfigured nics
      shell: |
        echo "do something"
`

	// 你期望替换的内容
	macTypes := map[string]string{
		"00:11:22:33:44:55": "IPV4_DHCP",
		"66:77:88:99:AA:BB": "NONE",
		"CC:DD:EE:FF:GG:HH": "IPV4",
	}

	// 构建 mac_types yaml 字符串
	var sb strings.Builder
	sb.WriteString("mac_types:\n")
	for mac, typ := range macTypes {
		sb.WriteString(fmt.Sprintf("      \"%s\": \"%s\"\n", mac, typ))
	}

	// 替换 mac_types: {}
	updated := strings.Replace(playbook, "mac_types: {}", sb.String(), 1)

	fmt.Println(updated)

	arr := []int{1, 2, 3, 4, 5, 6, 7}
	n := 1

	result := []int{}
	for i := len(arr) - n; i < len(arr) && i >= 0; i++ {
		result = append(result, arr[i])
	}

	fmt.Println("result:", result)
}
