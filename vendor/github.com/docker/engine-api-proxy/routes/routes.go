package routes

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

var (
	ContainerArchiveGet  = New(http.MethodGet, "/containers/{name:.*}/archive")
	ContainerArchiveHead = New(http.MethodHead, "/containers/{name:.*}/archive")
	ContainerArchivePut  = New(http.MethodPut, "/containers/{name:.*}/archive")
	ContainerAttach      = New(http.MethodPost, "/containers/{name:.*}/attach")
	ContainerAttachWS    = New(http.MethodGet, "/containers/{name:.*}/attach/ws")
	ContainerChanges     = New(http.MethodGet, "/containers/{name:.*}/changes")
	ContainerCommit      = New(http.MethodPost, "/commit")
	ContainerCreate      = New(http.MethodPost, "/containers/create")
	ContainerExec        = New(http.MethodGet, "/exec/{id:.*}/json")
	ContainerExecCreate  = New(http.MethodPost, "/containers/{name:.*}/exec")
	ContainerExecResize  = New(http.MethodPost, "/exec/{id:.*}/resize")
	ContainerExecStart   = New(http.MethodPost, "/exec/{id:.*}/start")
	ContainerExport      = New(http.MethodGet, "/containers/{name:.*}/export")
	ContainerInspect     = New(http.MethodGet, "/containers/{name:.*}/json")
	ContainerKill        = New(http.MethodPost, "/containers/{name:.*}/kill")
	ContainerList        = New(http.MethodGet, "/containers/json")
	ContainerLogs        = New(http.MethodGet, "/containers/{name:.*}/logs")
	ContainerPause       = New(http.MethodPost, "/containers/{name:.*}/pause")
	ContainerPrune       = New(http.MethodPost, "/containers/prune")
	ContainerRemove      = New(http.MethodDelete, "/containers/{name:.*}")
	ContainerRename      = New(http.MethodPost, "/containers/{name:.*}/rename")
	ContainerResize      = New(http.MethodPost, "/containers/{name:.*}/resize")
	ContainerRestart     = New(http.MethodPost, "/containers/{name:.*}/restart")
	ContainerStart       = New(http.MethodPost, "/containers/{name:.*}/start")
	ContainerStats       = New(http.MethodGet, "/containers/{name:.*}/stats")
	ContainerStop        = New(http.MethodPost, "/containers/{name:.*}/stop")
	ContainerTop         = New(http.MethodGet, "/containers/{name:.*}/top")
	ContainerUnpause     = New(http.MethodPost, "/containers/{name:.*}/unpause")
	ContainerUpdate      = New(http.MethodPost, "/containers/{name:.*}/update")
	ContainerWait        = New(http.MethodPost, "/containers/{name:.*}/wait")

	VolumeList    = New(http.MethodGet, "/volumes")
	VolumeInspect = New(http.MethodGet, "/volumes/{name:.*}")
	VolumeCreate  = New(http.MethodPost, "/volumes/create")
	VolumeRemove  = New(http.MethodDelete, "/volumes/{name:.*}")

	NetworkList       = New(http.MethodGet, "/networks")
	NetworkInspect    = New(http.MethodGet, "/networks/{name:.*}")
	NetworkCreate     = New(http.MethodPost, "/networks/create")
	NetworkRemove     = New(http.MethodDelete, "/networks/{name:.*}")
	NetworkConnect    = New(http.MethodPost, "/networks/{name:.*}/connect")
	NetworkDisconnect = New(http.MethodPost, "/networks/{name:.*}/disconnect")

	Events      = New(http.MethodGet, "/events")
	ImageBuild  = New(http.MethodPost, "/build")
	ImageCreate = New(http.MethodPost, "/images/create")
	ImagePush   = New(http.MethodPost, "/images/{name:.*}/push")
	PluginPull  = New(http.MethodPost, "/plugins/pull")
	PluginPush  = New(http.MethodPost, "/plugins/{name:.*}/push")

	ServiceList    = New(http.MethodGet, "/services")
	ServiceCreate  = New(http.MethodPost, "/services/create")
	ServiceInspect = New(http.MethodGet, "/services/{name:.*}")
	ServiceRemove  = New(http.MethodDelete, "/services/{name:.*}")
	ServiceUpdate  = New(http.MethodPost, "/services/{name:.*}/update")
	ServiceLogs    = New(http.MethodGet, "/services/{name:.*}/logs")

	SecretList    = New(http.MethodGet, "/secrets")
	SecretCreate  = New(http.MethodPost, "/secrets/create")
	SecretInspect = New(http.MethodGet, "/secrets/{name:.*}")
	SecretRemove  = New(http.MethodDelete, "/secrets/{name:.*}")
	SecretUpdate  = New(http.MethodPost, "/secrets/{name:.*}/update")
)

// Route is a data object which holds details about a route
type Route struct {
	Method string
	Path   string
}

// AsMuxRoute returns a mux.Route from the values set in this route
func (r *Route) AsMuxRoute() *mux.Route {
	route := &mux.Route{}
	return route.Methods(r.Method).Path(r.Path)
}

// Versioned returns a copy of this Route with a version prefixed Path
func (r *Route) Versioned() *Route {
	newRoute := *r
	newRoute.Path = "/v{version:[0-9.]+}" + r.Path
	return &newRoute
}

func (r *Route) String() string {
	return fmt.Sprintf("%s %s", r.Method, r.Path)
}

// New creates a new route from an HTTP method and a path
func New(method string, path string) *Route {
	return &Route{Method: method, Path: path}
}
