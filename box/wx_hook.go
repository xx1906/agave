package box

import (
	"bytes"
	"context"

	"fmt"
	"github.com/guzzsek/agave/encoding/json"
	"github.com/guzzsek/agave/ject"
	"io/ioutil"
	"net/http"
)

// 微信通知钩子的实现
type wechatMarkdownWebHook struct {
	WebHook string `json:"web_hook"`
	Msgtype string `json:"msgtype"`
}

func (c wechatMarkdownWebHook) Send(ctx context.Context, entry *ject.Entry) error {
	// 微信 markdown 通知类型
	type WxMarkdownContent struct {
		Msgtype  string                 `json:"msgtype"`
		Markdown map[string]interface{} `json:"markdown"`
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	v := WxMarkdownContent{Msgtype: "markdown", Markdown: make(map[string]interface{})}
	v.Markdown["content"] = fmt.Sprintf(`%s`, string(data))

	data, err = json.Marshal(v)
	buffer := bytes.NewBuffer(data)

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.WebHook, buffer)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	if data, err = ioutil.ReadAll(resp.Body); err != nil {
		return err
	}

	fmt.Println(string(data))
	return nil
}

func (c wechatMarkdownWebHook) Fire(ctx context.Context, entry *ject.Entry) error {

	return c.Send(ctx, entry)
}

func NewWechatMarkdownWebHook(webHook string) *wechatMarkdownWebHook {

	return &wechatMarkdownWebHook{WebHook: webHook, Msgtype: "markdown"}
}
