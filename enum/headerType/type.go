package headerType

type Type string

const (
	NewNode        Type = "NEW"
	NewChannel     Type = "NEW CHANNEL"
	UpdateChannel  Type = "UPDATE CHANNEL"
	ChannelInfo    Type = "CHANNEL INFO"
	PrivateMessage Type = "PM"
	Exit           Type = "EXIT"
	ChatMessage    Type = "CHAT MESSAGE"
)
