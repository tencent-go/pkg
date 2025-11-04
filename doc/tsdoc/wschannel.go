package tsdoc

import (
	"bytes"
	"path"
	"text/template"

	"github.com/tencent-go/pkg/doc/wsdoc"
	"github.com/tencent-go/pkg/util"
	"github.com/sirupsen/logrus"
)

func NewEventFile(subscribers []wsdoc.EventChannel, publishers []wsdoc.EventChannel, parentDir ...string) []util.DataFile {
	temp := `
import { remoteEventBus } from '@tencent-app/ts-event-bus';

/**
 * subscriber channels
 */
{{range .Listeners}}
{{- if .Description}}
// {{.Description}}
{{end}}
export const {{.FuncName}} = remoteEventBus.subscriberChannel<{{.Type}}>('{{.Topic}}');
{{end}}
/**
 * publisher channels
 */
{{range .Emitters}}
{{- if .Description}}
// {{.Description}}
{{end}}
export const {{.FuncName}} = remoteEventBus.publisherChannel<{{.Type}}>('{{.Topic}}');
{{end}}
`
	t, err := template.New("typescript_events").Parse(temp)
	if err != nil {
		logrus.Fatalf("parse interface template failed: %v", err)
	}
	data := eventData{
		Emitters:  make([]eventFunc, len(publishers)),
		Listeners: make([]eventFunc, len(subscribers)),
	}
	for i, d := range subscribers {
		data.Listeners[i] = newEventFunc(d)
	}
	for i, d := range publishers {
		data.Emitters[i] = newEventFunc(d)
	}
	var buf bytes.Buffer
	if err = t.Execute(&buf, data); err != nil {
		logrus.Fatalf("execute interface template failed: %v", err)
	}
	return []util.DataFile{
		{
			Dir:  path.Join(parentDir...),
			Name: "events.ts",
			Data: buf.Bytes(),
		},
	}
}

type eventData struct {
	Emitters  []eventFunc
	Listeners []eventFunc
}

func newEventFunc(doc wsdoc.EventChannel) eventFunc {
	res := eventFunc{
		Description: doc.Description,
		Topic:       doc.Topic,
		Type:        "undefined",
	}
	if doc.Type != nil {
		res.Type = parseType(*doc.Type, nil)
	}
	res.FuncName = convertName(doc.Topic)
	return res
}

type eventFunc struct {
	FuncName    string
	Type        string
	Topic       string
	Description string
}
