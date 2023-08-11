package capitalws

import (
	"encoding/json"
	"golang.org/x/exp/maps"
	_ "strings"

	"github.com/chuanfoo/capital-go/websocket/models"
)

// subscriptions stores topic subscriptions for resubscribing after disconnect.
type subscriptions map[Topic]set
type set map[string]struct{}

// add inserts a set of tickers to the given topic.
func (subs subscriptions) add(topic Topic, tickers ...string) {
	_, prefixExists := subs[topic]
	if !prefixExists {
		subs[topic] = make(set)
	}
	for _, t := range tickers {
		subs[topic][t] = struct{}{}
	}
}

// get retrieves a list of subscription messages based on what has been cached.
func (subs subscriptions) get(cst, securityToken string) []json.RawMessage {
	var msgs []json.RawMessage
	for topic, tickers := range subs {
		msg, err := getSub(topic, cst, securityToken, maps.Keys(tickers)...)
		if err != nil {
			continue // skip malformed messages
		}
		msgs = append(msgs, msg)
	}
	return msgs
}

// delete removes a set of tickers from the given topic.
func (subs subscriptions) delete(topic Topic, tickers ...string) {
	for _, t := range tickers {
		delete(subs[topic], t)
	}
	if len(subs[topic]) == 0 {
		delete(subs, topic)
	}
}

// getSub builds a subscription message for a given topic.
func getSub(topic Topic, cst, securityToken string, epics ...string) (json.RawMessage, error) {
	msg, err := json.Marshal(&models.OHLCMarketDataSubscribe{
		Destination:   topic.Destination(),
		CorrelationID: topic.CorrelationID(),
		Cst:           cst,
		SecurityToken: securityToken,
		Payload:       models.Payload{Epics: epics, Type: "classic", Resolutions: []string{"MINUTE"}},
	})
	if err != nil {
		return nil, err
	}

	return msg, nil
}
