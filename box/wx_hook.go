package box

import (
	"bytes"
	"context"
	"os"

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

// 需要参照微信机器人的通知配置
func (c wechatMarkdownWebHook) Send(ctx context.Context, entry *ject.Entry) error {
	// 微信 markdown 通知类型
	type WxMarkdownContent struct {
		Msgtype  string                 `json:"msgtype"`
		Markdown map[string]interface{} `json:"markdown"`
	}

	var (
		request *http.Request  // 请求体
		resp    *http.Response // 响应体

		err  error
		data []byte

		v WxMarkdownContent
	)

	if data, err = json.Marshal(entry); err != nil {
		return err
	}

	v = WxMarkdownContent{Msgtype: "markdown", Markdown: make(map[string]interface{})}
	v.Markdown["content"] = fmt.Sprintf(`%s`, string(data))

	data, err = json.Marshal(v)
	buffer := bytes.NewBuffer(data)

	// 构造请求体
	if request, err = http.NewRequestWithContext(ctx, http.MethodPost, c.WebHook, buffer); err != nil {
		return err
	}

	if resp, err = http.DefaultClient.Do(request); err != nil {
		return err
	}

	defer resp.Body.Close()
	if data, err = ioutil.ReadAll(resp.Body); err != nil {
		return err
	}

	fmt.Fprintln(os.Stdout, string(data))
	return nil
}

func (c wechatMarkdownWebHook) Fire(ctx context.Context, entry *ject.Entry) error {

	return c.Send(ctx, entry)
}

func NewWechatMarkdownWebHook(webHook string) *wechatMarkdownWebHook {

	return &wechatMarkdownWebHook{WebHook: webHook, Msgtype: "markdown"}
}
