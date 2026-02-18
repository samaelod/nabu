local config = {}

-- GLOBALS ----------------------------------------
config.globals = {
	protocol = "tcp", -- Only Tcp for now
	play_mode = "pcap", -- pcap/live
	timeout = 5000, -- in milliseconds
	delay = 100, -- in milliseconds
}

config.endpoints = {
	{
		id = 0,
		kind = "server",
		address = "127.0.0.1",
		port = 9990,
	},
	{
		id = 1,
		kind = "client",
		address = "127.0.0.1",
		port = 42069,
	},
	{
		id = 2,
		kind = "server",
		address = "127.0.0.1",
		port = 9990,
	},
	{
		id = 4,
		kind = "client",
		address = "127.0.0.1",
		port = 42069,
	},
}

-- MESSAGES ----------------------------------------
config.messages = {
	-- Message 0: SYN --
	{
		from = 1,
		to = 0,
		kind = "syn",
		value = "", 
		t_delta = 0, 
	},

	-- Message 1: SYN-ACK --
	{
		from = 0,
		to = 1,
		kind = "syn-ack",
		value = "", 
		t_delta = 50, 
	},

	-- Message 2: ACK --
	{
		from = 1,
		to = 0,
		kind = "ack",
		value = "", 
		t_delta = 50, 
	},
	-- Message 3: Data --
	{
		from = 0,
		to = 1,
		kind = "data",
		value = "0123456789ABCDEF", 
		t_delta = 100, 
	},
}

return config
