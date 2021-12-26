package pixiv

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/RicheyJang/PaimengBot/utils"
	"github.com/RicheyJang/PaimengBot/utils/client"
	"github.com/RicheyJang/PaimengBot/utils/consts"
	"github.com/RicheyJang/PaimengBot/utils/images"

	"github.com/wdvxdr1123/ZeroBot/message"
)

// 生成单条Pixiv消息
func genSinglePicMsg(pic *PictureInfo) (message.Message, error) {
	// 初始化
	if pic == nil {
		return nil, fmt.Errorf("pic is nil")
	}
	if len(pic.URL) == 0 {
		err := pic.getURLByPID()
		if err != nil {
			return nil, err
		}
	}
	// 下载图片
	path, err := images.GetNewTempSavePath("pixiv")
	if err != nil {
		return nil, err
	}
	c := client.NewHttpClient(&client.HttpOptions{TryTime: 2})
	err = c.DownloadToFile(path, pic.URL)
	if err != nil {
		return nil, err
	}
	// 构成消息
	picMsg, err := utils.GetImageFileMsg(path)
	if err != nil {
		return nil, err
	}
	// 文字
	var tags []string
	for _, tag := range pic.Tags {
		if isCNOrEn(tag) {
			tags = append(tags, tag)
		}
	}
	tip := fmt.Sprintf("PID: %v", pic.PID)
	if pic.P != 0 {
		tip += fmt.Sprintf("(p%d)", pic.P)
	}
	if pic.UID != 0 {
		tip += fmt.Sprintf("\nUID: %v", pic.UID)
	}
	if len(tags) > 0 {
		tip += fmt.Sprintf("\n标签: %v", strings.Join(tags, ","))
	}
	return message.Message{message.Text(pic.Title), picMsg, message.Text(tip)}, nil
}

func (pic *PictureInfo) getURLByPID() (err error) {
	if pic.PID == 0 {
		return fmt.Errorf("pid is 0")
	}
	// 整理API URL
	api := proxy.GetAPIConfig(consts.APIOfHibiAPIKey)
	if len(api) == 0 {
		return fmt.Errorf("API of HibiAPI is empty")
	}
	if !strings.HasPrefix(api, "http://") && !strings.HasPrefix(api, "https://") {
		api = "https://" + api
	}
	if !strings.HasSuffix(api, "/") {
		api += "/"
	}
	api = fmt.Sprintf("%sapi/pixiv/illust?id=%v", api, pic.PID)
	// 调用
	c := client.NewHttpClient(nil)
	rsp, err := c.GetGJson(api)
	if err != nil {
		return err
	}
	rsp = rsp.Get("illust")
	if !rsp.Exists() {
		return fmt.Errorf("illust is not found")
	}
	defer func() { // 替换代理
		if err == nil && len(pic.URL) == 0 {
			err = fmt.Errorf("unexpected error")
		} else if len(pic.URL) > 0 {
			p := proxy.GetConfigString("proxy")
			if len(p) > 0 {
				pic.URL = strings.ReplaceAll(pic.URL, "i.pximg.net", p)
				pic.URL = strings.ReplaceAll(pic.URL, "i.pixiv.net", p)
			}
		}
	}()
	// 解析
	if rsp.Get("page_count").Int() == 1 {
		pic.URL = rsp.Get("meta_single_page.original_image_url").String()
	} else if rsp.Get("page_count").Int() > int64(pic.P) {
		pic.URL = rsp.Get("meta_pages." + strconv.Itoa(pic.P) + ".image_urls.original").String()
	}
	return nil
}

var asciiReg = regexp.MustCompile(`^[A-Za-z0-9_+,=~!@#<>\[\]{}:/.?'"$%&*()\-\\]+$`)

func isCNOrEn(str string) bool {
	for _, c := range str {
		if 0x3040 <= c && c <= 0x31FF { // 日语
			return false
		}
		if 0xAC00 <= c && c <= 0xD7AF { // 韩语
			return false
		}
	}
	for _, c := range str {
		if 0x4E00 <= c && c <= 0x9FD5 { // 有中文
			return true
		}
	}
	if asciiReg.MatchString(str) {
		return true
	}
	return false
}
