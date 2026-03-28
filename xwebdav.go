package webdav

import (
	"context"
	"errors"
	"net/http"

	"github.com/emersion/go-webdav/internal"
)

// PHAssetHandler 处理 Apple 照片资产的 PROPFIND 请求。
//
// 采用 go-webdav 库实现是因为它提供了 Backend 接口，
// 可以完全控制 PROPFIND 响应，从而实现"以 Asset 为单位返回信息"的需求。
// 标准库 golang.org/x/net/webdav 的定制能力不足以支持这个需求。
//
// 上层 backupWebDAVHandler 根据请求特征决定是否分流到此处。
// 仅 Apple Photo 备份的 /current 或 /archive 目录的 PROPFIND 请求会到达这里。
//
// Asset 识别逻辑通过 Resolver 注入，避免 go-webdav 反向依赖业务仓。
//
// [注]：internal包无法在外部使用，所以需要在这里实现一个 Handler。
type PHAssetHandler struct {
	FileSystem FileSystem
	Resolver   PHAssetResolver
}

func (h *PHAssetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.FileSystem == nil {
		http.Error(w, "xwebdav: no filesystem available", http.StatusInternalServerError)
		return
	}
	if h.Resolver == nil {
		http.Error(w, "xwebdav: no resolver available", http.StatusInternalServerError)
		return
	}

	b := backendPHA{
		backend:  &backend{h.FileSystem},
		resolver: h.Resolver,
	}
	hh := internal.Handler{Backend: &b}
	hh.ServeHTTP(w, r)
}

// PHAssetResolver abstracts business-specific path/asset parsing logic.
// Callers (e.g. libscm) should inject an implementation.
type PHAssetResolver interface {
	ResolvePrimary(ctx context.Context, fs FileSystem, scopePath string, child FileInfo) (*FileInfo, error)
}

type backendPHA struct {
	*backend
	resolver PHAssetResolver
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
			resolved, resolveErr := b.resolver.ResolvePrimary(r.Context(), b.FileSystem, r.URL.Path, child)
			if resolveErr != nil {
				return nil, resolveErr
			}
			if resolved != nil {
				item = resolved
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

// backendPHA 只实现了 PropFind 方法来处理 Apple 照片资产。
// 由上层 backupWebDAVHandler分流， PROPFIND 请求才会到达此处。
// 如下方法都不应该被调用，直接返回未实现错误。
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
