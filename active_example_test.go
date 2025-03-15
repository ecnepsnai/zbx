package zbx_test

import "github.com/ecnepsnai/zbx"

func ExampleStartActive() {
	session, items, err := zbx.StartActive(
		"myserver.example.com",     // The name of this agent as configured on the zabbix server
		"zabbix.example.com:10051", // The address to the zabbix server's listener - this is different from the web interface
	)
	if err != nil {
		panic(err)
	}

	// items contains a slice of what items the zabbix server expects you to send, and how frequently
	if len(items) == 0 {
		// If items is empty, then the zabbix server either does not recognize this host (myserver.example.com)
		// or there are no items with the "Zabbix Agent (Active)" type.
		panic("no items")
	}

	// Active checks work using internal item IDs, which you must map to the item key
	// For example, here we'll map the 'agent.ping' item to its id.
	var pingItemId int
	for _, item := range items {
		if item.Key == "agent.ping" {
			pingItemId = item.ItemId
			break
		}
	}
	if pingItemId == 0 {
		panic("item not found")
	}

	// You can send multiple item values at once, but in this example we'll only send one
	if err := session.Send(map[int]string{
		pingItemId: "ok",
	}); err != nil {
		panic(err)
	}
}
