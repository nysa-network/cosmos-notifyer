package main

type (
	Event struct {
		Type       string   `bson:"type"`
		Attributes []KvPair `bson:"attributes"`
	}

	KvPair struct {
		Key   string `bson:"key"`
		Value string `bson:"value"`
	}

	EventNew struct {
		MsgIndex int     `bson:"msg_index" json:"msg_index"`
		Events   []Event `bson:"events"`
	}
)
