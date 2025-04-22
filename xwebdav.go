package webdav

import (
	"errors"
	"libscm/util"
	"net/http"
	"path"

	"github.com/emersion/go-webdav/internal"
)

// 自定义的webdav扩展Handler，处理定制逻辑，internal包无法在外部使用，所以需要在这里实现
type PHAssetHandler struct {
	FileSystem FileSystem
}

func (h *PHAssetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.FileSystem == nil {
		http.Error(w, "xwebdav: no filesystem available", http.StatusInternalServerError)
		return
	}

	b := backendPHA{&backend{h.FileSystem}}
	hh := internal.Handler{Backend: &b}
	hh.ServeHTTP(w, r)
}

type backendPHA struct {
	*backend
}

// 实现Apple 备份照片的浏览逻辑，以Asset为单位返回信息
func (b *backendPHA) PropFind(r *http.Request, propfind *internal.PropFind, depth internal.Depth) (*internal.MultiStatus, error) {
	fi, err := b.FileSystem.Stat(r.Context(), r.URL.Path)
	if err != nil {
		return nil, err
	}

	// 参数校验
	if !fi.IsDir || depth != internal.DepthOne || (r.URL.Path != "/current" && r.URL.Path != "/archive") {
		return nil, errors.New("xwebdav: invalid prop find paramters")
	}

	children, err := b.FileSystem.ReadDir(r.Context(), r.URL.Path, false)
	if err != nil {
		return nil, err
	}

	resps := make([]internal.Response, len(children))
	for i, child := range children {
		item := &child
		if child.IsDir && child.Path != r.URL.Path {
			// 查找渲染的主资产信息
			// 目录名中携带了类型信息，直接通过目录名判断是否视频or图片
			found := false
			if util.IsVideoFile(child.Path) {
				item, err = b.FileSystem.Stat(r.Context(), path.Join(child.Path, "FullSizeRender.mov"))
				found = err == nil
			} else if util.IsImageFile(child.Path) {
				item, err = b.FileSystem.Stat(r.Context(), path.Join(child.Path, "FullSizeRender.jpg"))
				found = err == nil
			}

			// 未找到渲染信息，解析名称, 获取主资产名
			if !found {
				_, name, err := util.ParseApplePHAssetName(path.Base(child.Path))
				if err != nil && r.URL.Path == "/archive" {
					// 解析失败，尝试从archive目录名中获取原始名称
					name = util.RetrieveOriginalNameFromApplePHAssetArchiveName(path.Base(child.Path))
				}

				if name != "" {
					// 通过名称获取主资产信息
					item, _ = b.FileSystem.Stat(r.Context(), path.Join(child.Path, name))
				}
			}

			if item == nil {
				item = &child // 没有找到可用信息，继续使用原始信息吧
			}
		}

		resp, err := b.propFindFile(propfind, item)
		if err != nil {
			return nil, err
		}

		resps[i] = *resp
	}

	return internal.NewMultiStatus(resps...), nil
}

// 如下方法都不实现，使用默认的实现
func (b *backendPHA) Options(r *http.Request) (caps []string, allow []string, err error) {
	return nil, nil, errors.New("xwebdav: Options not implemented")
}

func (b *backendPHA) HeadGet(w http.ResponseWriter, r *http.Request) error {
	return errors.New("xwebdav: HeadGet not implemented")
}

func (b *backendPHA) PropPatch(r *http.Request, pu *internal.PropertyUpdate) (*internal.Response, error) {
	return nil, errors.New("xwebdav: PropPatch not implemented")
}

func (b *backendPHA) Put(w http.ResponseWriter, r *http.Request) error {
	return errors.New("xwebdav: Put not implemented")
}

func (b *backendPHA) Delete(r *http.Request) error {
	return errors.New("xwebdav: Delete not implemented")
}

func (b *backendPHA) Mkcol(r *http.Request) error {
	return errors.New("xwebdav: Mkcol not implemented")
}

func (b *backendPHA) Copy(r *http.Request, dest *internal.Href, recursive, overwrite bool) (created bool, err error) {
	return false, errors.New("xwebdav: Copy not implemented")
}

func (b *backendPHA) Move(r *http.Request, dest *internal.Href, overwrite bool) (created bool, err error) {
	return false, errors.New("xwebdav: Move not implemented")
}
