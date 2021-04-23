// @desc:   gin 奔溃拦截器, 主要用于报告程序 panic 的具体信息
// @author: 小肥
// @date:   2021年04月23日
// @email:  <2356450144@qq.com>

package ject

import "context"

// 回调接口 interface
type Hook interface {
	Fire(ctx context.Context, entry *Entry) error
}
