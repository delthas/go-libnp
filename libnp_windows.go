package libnp

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"github.com/go-ole/go-ole"
	"golang.org/x/sys/windows"
)

type IVectorView struct {
	ole.IInspectable
}

type IVectorViewVtbl struct {
	ole.IInspectableVtbl

	GetAt   uintptr
	GetSize uintptr
	IndexOf uintptr
	GetMany uintptr
}

func (v *IVectorView) VTable() *IVectorViewVtbl {
	return (*IVectorViewVtbl)(unsafe.Pointer(v.RawVTable))
}

func (v *IVectorView) GetAt(index uint32) (unsafe.Pointer, error) {
	var out unsafe.Pointer
	hr, _, _ := syscall.SyscallN(
		v.VTable().GetAt,
		uintptr(unsafe.Pointer(v)),
		uintptr(index),
		uintptr(unsafe.Pointer(&out)),
	)
	if hr != 0 {
		return nil, ole.NewError(hr)
	}
	return out, nil
}

func (v *IVectorView) GetSize() (uint32, error) {
	var out uint32
	hr, _, _ := syscall.SyscallN(
		v.VTable().GetSize,
		uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(&out)),
	)
	if hr != 0 {
		return 0, ole.NewError(hr)
	}
	return out, nil
}

func (v *IVectorView) IndexOf(value unsafe.Pointer) (uint32, bool, error) {
	var index uint32
	var out bool
	hr, _, _ := syscall.SyscallN(
		v.VTable().IndexOf,
		uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(&value)),
		uintptr(unsafe.Pointer(&index)),
		uintptr(unsafe.Pointer(&out)),
	)
	if hr != 0 {
		return 0, false, ole.NewError(hr)
	}
	return index, out, nil
}

func (v *IVectorView) GetMany(startIndex uint32, itemsSize uint32) ([]unsafe.Pointer, uint32, error) {
	var items = make([]unsafe.Pointer, itemsSize)
	var out uint32
	hr, _, _ := syscall.SyscallN(
		v.VTable().GetMany,
		uintptr(unsafe.Pointer(v)),
		uintptr(startIndex),
		uintptr(itemsSize),
		uintptr(unsafe.Pointer(&items[0])),
		uintptr(unsafe.Pointer(&out)),
	)
	if hr != 0 {
		return nil, 0, ole.NewError(hr)
	}
	return items, out, nil
}

// Only a limited number of callbacks may be created in a single Go process,
// and any memory allocated for these callbacks is never released.
// Between NewCallback and NewCallbackCDecl, at least 1024 callbacks can always be created.
var (
	queryInterfaceCallback = syscall.NewCallback(queryInterface)
	addRefCallback         = syscall.NewCallback(addRef)
	releaseCallback        = syscall.NewCallback(release)
	invokeCallback         = syscall.NewCallback(invoke)
)

// Delegate represents a WinRT delegate class.
type Delegate interface {
	GetIID() *ole.GUID
	Invoke(instancePtr, rawArgs0, rawArgs1, rawArgs2, rawArgs3, rawArgs4, rawArgs5, rawArgs6, rawArgs7, rawArgs8 unsafe.Pointer) uintptr
	AddRef() uint64
	Release() uint64
}

// Callbacks contains the syscalls registered on Windows.
type Callbacks struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
	Invoke         uintptr
}

var mutex = sync.RWMutex{}
var instances = make(map[uintptr]Delegate)

// RegisterCallbacks adds the given pointer and the Delegate it points to to our instances.
// This is required to redirect received callbacks to the correct object instance.
// The function returns the callbacks to use when creating a new delegate instance.
func RegisterCallbacks(ptr unsafe.Pointer, inst Delegate) *Callbacks {
	mutex.Lock()
	defer mutex.Unlock()
	instances[uintptr(ptr)] = inst

	return &Callbacks{
		QueryInterface: queryInterfaceCallback,
		AddRef:         addRefCallback,
		Release:        releaseCallback,
		Invoke:         invokeCallback,
	}
}

func getInstance(ptr unsafe.Pointer) (Delegate, bool) {
	mutex.RLock() // locks writing, allows concurrent read
	defer mutex.RUnlock()

	i, ok := instances[uintptr(ptr)]
	return i, ok
}

func removeInstance(ptr unsafe.Pointer) {
	mutex.Lock()
	defer mutex.Unlock()
	delete(instances, uintptr(ptr))
}

func queryInterface(instancePtr unsafe.Pointer, iidPtr unsafe.Pointer, ppvObject *unsafe.Pointer) uintptr {
	instance, ok := getInstance(instancePtr)
	if !ok {
		// instance not found
		return ole.E_POINTER
	}

	// Checkout these sources for more information about the QueryInterface method.
	//   - https://docs.microsoft.com/en-us/cpp/atl/queryinterface
	//   - https://docs.microsoft.com/en-us/windows/win32/api/unknwn/nf-unknwn-iunknown-queryinterface(refiid_void)

	if ppvObject == nil {
		// If ppvObject (the address) is nullptr, then this method returns E_POINTER.
		return ole.E_POINTER
	}

	// This function must adhere to the QueryInterface defined here:
	// https://docs.microsoft.com/en-us/windows/win32/api/unknwn/nn-unknwn-iunknown
	if iid := (*ole.GUID)(iidPtr); ole.IsEqualGUID(iid, instance.GetIID()) || ole.IsEqualGUID(iid, ole.IID_IUnknown) || ole.IsEqualGUID(iid, ole.IID_IInspectable) {
		*ppvObject = instancePtr
	} else {
		*ppvObject = nil
		// Return E_NOINTERFACE if the interface is not supported
		return ole.E_NOINTERFACE
	}

	// If the COM object implements the interface, then it returns
	// a pointer to that interface after calling IUnknown::AddRef on it.
	(*ole.IUnknown)(*ppvObject).AddRef()

	// Return S_OK if the interface is supported
	return ole.S_OK
}

func invoke(instancePtr, rawArgs0, rawArgs1, rawArgs2, rawArgs3, rawArgs4, rawArgs5, rawArgs6, rawArgs7, rawArgs8 unsafe.Pointer) uintptr {
	instance, ok := getInstance(instancePtr)
	if !ok {
		// instance not found
		return ole.E_FAIL
	}

	return instance.Invoke(instancePtr, rawArgs0, rawArgs1, rawArgs2, rawArgs3, rawArgs4, rawArgs5, rawArgs6, rawArgs7, rawArgs8)
}

func addRef(instancePtr unsafe.Pointer) uint64 {
	instance, ok := getInstance(instancePtr)
	if !ok {
		// instance not found
		return ole.E_FAIL
	}

	return instance.AddRef()
}

func release(instancePtr unsafe.Pointer) uint64 {
	instance, ok := getInstance(instancePtr)
	if !ok {
		// instance not found
		return ole.E_FAIL
	}

	rem := instance.Release()
	if rem == 0 {
		// remove this delegate
		removeInstance(instancePtr)
	}
	return rem
}

type (
	heapHandle = uintptr
	win32Error uint32
	heapFlags  uint32
)

const (
	heapNone       heapFlags = 0
	heapZeroMemory heapFlags = 8 // The allocated memory will be initialized to zero.
)

var (
	libKernel32 = windows.NewLazySystemDLL("kernel32.dll")

	pHeapFree       uintptr
	pHeapAlloc      uintptr
	pGetProcessHeap uintptr

	hHeap heapHandle
)

func init() {
	hHeap, _ = getProcessHeap()
}

// https://docs.microsoft.com/en-us/windows/win32/api/heapapi/nf-heapapi-heapalloc
func heapAlloc(hHeap heapHandle, dwFlags heapFlags, dwBytes uintptr) unsafe.Pointer {
	addr := getProcAddr(&pHeapAlloc, libKernel32, "HeapAlloc")
	allocatedPtr, _, _ := syscall.SyscallN(addr, hHeap, uintptr(dwFlags), dwBytes)
	// Since this pointer is allocated in the heap by Windows, it will never be
	// GCd by Go, so this is a safe operation.
	// But linter thinks it is not (probably because we are not using CGO) and fails.
	//goland:noinspection GoVetUnsafePointer
	return unsafe.Pointer(allocatedPtr) //nolint:gosec,govet
}

// https://docs.microsoft.com/en-us/windows/win32/api/heapapi/nf-heapapi-heapfree
func heapFree(hHeap heapHandle, dwFlags heapFlags, lpMem unsafe.Pointer) (bool, win32Error) {
	addr := getProcAddr(&pHeapFree, libKernel32, "HeapFree")
	ret, _, err := syscall.SyscallN(addr, hHeap, uintptr(dwFlags), uintptr(lpMem))
	return ret == 0, win32Error(err)
}

func getProcessHeap() (heapHandle, win32Error) {
	addr := getProcAddr(&pGetProcessHeap, libKernel32, "GetProcessHeap")
	ret, _, err := syscall.SyscallN(addr)
	return ret, win32Error(err)
}

func getProcAddr(pAddr *uintptr, lib *windows.LazyDLL, procName string) uintptr {
	addr := atomic.LoadUintptr(pAddr)
	if addr == 0 {
		addr = lib.NewProc(procName).Addr()
		atomic.StoreUintptr(pAddr, addr)
	}
	return addr
}

type AsyncStatus int32

const (
	AsyncStatusCanceled  AsyncStatus = 2
	AsyncStatusCompleted AsyncStatus = 1
	AsyncStatusError     AsyncStatus = 3
	AsyncStatusStarted   AsyncStatus = 0
)

type AsyncOperationCompletedHandler struct {
	ole.IUnknown
	sync.Mutex
	refs uint64
	IID  ole.GUID
}

type AsyncOperationCompletedHandlerVtbl struct {
	ole.IUnknownVtbl
	Invoke uintptr
}

type AsyncOperationCompletedHandlerCallback func(instance *AsyncOperationCompletedHandler, asyncInfo *IAsyncOperation[any], asyncStatus AsyncStatus)

var callbacksAsyncOperationCompletedHandler = &asyncOperationCompletedHandlerCallbacks{
	mu:        &sync.Mutex{},
	callbacks: make(map[unsafe.Pointer]AsyncOperationCompletedHandlerCallback),
}

var releaseChannelsAsyncOperationCompletedHandler = &asyncOperationCompletedHandlerReleaseChannels{
	mu:    &sync.Mutex{},
	chans: make(map[unsafe.Pointer]chan struct{}),
}

func NewAsyncOperationCompletedHandler(iid *ole.GUID, callback AsyncOperationCompletedHandlerCallback) *AsyncOperationCompletedHandler {
	size := unsafe.Sizeof(*(*AsyncOperationCompletedHandler)(nil))
	instPtr := heapAlloc(hHeap, heapZeroMemory, size)
	inst := (*AsyncOperationCompletedHandler)(instPtr)

	callbacks := RegisterCallbacks(instPtr, inst)

	// Initialize all properties: the malloc may contain garbage
	inst.RawVTable = (*interface{})(unsafe.Pointer(&AsyncOperationCompletedHandlerVtbl{
		IUnknownVtbl: ole.IUnknownVtbl{
			QueryInterface: callbacks.QueryInterface,
			AddRef:         callbacks.AddRef,
			Release:        callbacks.Release,
		},
		Invoke: callbacks.Invoke,
	}))
	inst.IID = *iid // copy contents
	inst.Mutex = sync.Mutex{}
	inst.refs = 0

	callbacksAsyncOperationCompletedHandler.add(unsafe.Pointer(inst), callback)

	// See the docs in the releaseChannelsAsyncOperationCompletedHandler struct
	releaseChannelsAsyncOperationCompletedHandler.acquire(unsafe.Pointer(inst))

	inst.addRef()
	return inst
}

func (h *AsyncOperationCompletedHandler) GetIID() *ole.GUID {
	return &h.IID
}

// addRef increments the reference counter by one
func (h *AsyncOperationCompletedHandler) addRef() uint64 {
	h.Lock()
	defer h.Unlock()
	h.refs++
	return h.refs
}

// removeRef decrements the reference counter by one. If it was already zero, it will just return zero.
func (h *AsyncOperationCompletedHandler) removeRef() uint64 {
	h.Lock()
	defer h.Unlock()

	if h.refs > 0 {
		h.refs--
	}

	return h.refs
}

func (h *AsyncOperationCompletedHandler) Invoke(instancePtr, rawArgs0, rawArgs1, rawArgs2, rawArgs3, rawArgs4, rawArgs5, rawArgs6, rawArgs7, rawArgs8 unsafe.Pointer) uintptr {
	asyncInfoPtr := rawArgs0
	asyncStatusRaw := (int32)(uintptr(rawArgs1))

	// See the quote above.
	asyncInfo := (*IAsyncOperation[any])(asyncInfoPtr)
	asyncStatus := (AsyncStatus)(asyncStatusRaw)
	if callback, ok := callbacksAsyncOperationCompletedHandler.get(instancePtr); ok {
		callback(h, asyncInfo, asyncStatus)
	}
	return ole.S_OK
}

func (h *AsyncOperationCompletedHandler) AddRef() uint64 {
	return h.addRef()
}

func (h *AsyncOperationCompletedHandler) Release() uint64 {
	rem := h.removeRef()
	if rem == 0 {
		// We're done.
		instancePtr := unsafe.Pointer(h)
		callbacksAsyncOperationCompletedHandler.delete(instancePtr)

		// stop release channels used to avoid
		// https://github.com/golang/go/issues/55015
		releaseChannelsAsyncOperationCompletedHandler.release(instancePtr)

		heapFree(hHeap, heapNone, instancePtr)
	}
	return rem
}

type asyncOperationCompletedHandlerCallbacks struct {
	mu        *sync.Mutex
	callbacks map[unsafe.Pointer]AsyncOperationCompletedHandlerCallback
}

func (m *asyncOperationCompletedHandlerCallbacks) add(p unsafe.Pointer, v AsyncOperationCompletedHandlerCallback) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.callbacks[p] = v
}

func (m *asyncOperationCompletedHandlerCallbacks) get(p unsafe.Pointer) (AsyncOperationCompletedHandlerCallback, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	v, ok := m.callbacks[p]
	return v, ok
}

func (m *asyncOperationCompletedHandlerCallbacks) delete(p unsafe.Pointer) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.callbacks, p)
}

// typedEventHandlerReleaseChannels keeps a map with channels
// used to keep a goroutine alive during the lifecycle of this object.
// This is required to avoid causing a deadlock error.
// See this: https://github.com/golang/go/issues/55015
type asyncOperationCompletedHandlerReleaseChannels struct {
	mu    *sync.Mutex
	chans map[unsafe.Pointer]chan struct{}
}

func (m *asyncOperationCompletedHandlerReleaseChannels) acquire(p unsafe.Pointer) {
	m.mu.Lock()
	defer m.mu.Unlock()

	c := make(chan struct{})
	m.chans[p] = c

	go func() {
		// we need a timer to trick the go runtime into
		// thinking there's still something going on here
		// but we are only really interested in <-c
		t := time.NewTimer(time.Minute)
		for {
			select {
			case <-t.C:
				t.Reset(time.Minute)
			case <-c:
				t.Stop()
				return
			}
		}
	}()
}

func (m *asyncOperationCompletedHandlerReleaseChannels) release(p unsafe.Pointer) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if c, ok := m.chans[p]; ok {
		close(c)
		delete(m.chans, p)
	}
}

type IAsyncOperation[V any] struct {
	ole.IInspectable
}

type IAsyncOperationVtbl struct {
	ole.IInspectableVtbl
	PutCompleted uintptr
	GetCompleted uintptr
	GetResults   uintptr
}

func (v *IAsyncOperation[V]) VTable() *IAsyncOperationVtbl {
	return (*IAsyncOperationVtbl)(unsafe.Pointer(v.RawVTable))
}

func (v *IAsyncOperation[V]) PutCompleted(handler *AsyncOperationCompletedHandler) error {
	hr, _, _ := syscall.SyscallN(
		v.VTable().PutCompleted,
		uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(handler)),
	)
	if hr != 0 {
		return ole.NewError(hr)
	}
	return nil
}

func (v *IAsyncOperation[V]) GetCompleted() (*AsyncOperationCompletedHandler, error) {
	var r *AsyncOperationCompletedHandler
	hr, _, _ := syscall.SyscallN(
		v.VTable().GetCompleted,
		uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(&r)),
	)
	if hr != 0 {
		return nil, ole.NewError(hr)
	}
	return r, nil
}

func (v *IAsyncOperation[V]) GetResults() (*V, error) {
	var r *V
	hr, _, _ := syscall.Syscall(
		v.VTable().GetResults,
		2,
		uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(&r)),
		0)
	if hr != 0 {
		return nil, ole.NewError(hr)
	}
	return r, nil
}

func (v *IAsyncOperation[V]) WaitResult(guid string) (*V, error) {
	ch := make(chan AsyncStatus, 1)
	handler := NewAsyncOperationCompletedHandler(ole.NewGUID(guid), func(instance *AsyncOperationCompletedHandler, asyncInfo *IAsyncOperation[any], asyncStatus AsyncStatus) {
		ch <- asyncStatus
	})
	defer handler.Release()
	if err := v.PutCompleted(handler); err != nil {
		return nil, err
	}
	status := <-ch
	if status != AsyncStatusCompleted {
		return nil, fmt.Errorf("async operation failed with status %d", status)
	}
	r, err := v.GetResults()
	if err != nil {
		return nil, err
	}
	return r, nil
}

type IGlobalSystemMediaTransportControlsSessionManagerStatics struct {
	ole.IInspectable
}

type IGlobalSystemMediaTransportControlsSessionManagerStaticsVtbl struct {
	ole.IInspectableVtbl
	RequestAsync uintptr
}

func (v *IGlobalSystemMediaTransportControlsSessionManagerStatics) VTable() *IGlobalSystemMediaTransportControlsSessionManagerStaticsVtbl {
	return (*IGlobalSystemMediaTransportControlsSessionManagerStaticsVtbl)(unsafe.Pointer(v.RawVTable))
}

func (v *IGlobalSystemMediaTransportControlsSessionManagerStatics) RequestAsync() (*IAsyncOperation[IGlobalSystemMediaTransportControlsSessionManager], error) {
	var r *IAsyncOperation[IGlobalSystemMediaTransportControlsSessionManager]
	hr, _, _ := syscall.Syscall(
		v.VTable().RequestAsync,
		2,
		uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(&r)),
		0)
	if hr != 0 {
		return nil, ole.NewError(hr)
	}
	return r, nil
}

type IGlobalSystemMediaTransportControlsSessionManager struct {
	ole.IInspectable
}

type IGlobalSystemMediaTransportControlsSessionManagerVtbl struct {
	ole.IInspectableVtbl
	GetCurrentSession           uintptr
	GetSessions                 uintptr
	AddCurrentSessionChanged    uintptr
	RemoveCurrentSessionChanged uintptr
	AddSessionsChanged          uintptr
	RemoveSessionsChanged       uintptr
}

func (v *IGlobalSystemMediaTransportControlsSessionManager) VTable() *IGlobalSystemMediaTransportControlsSessionManagerVtbl {
	return (*IGlobalSystemMediaTransportControlsSessionManagerVtbl)(unsafe.Pointer(v.RawVTable))
}

func (v *IGlobalSystemMediaTransportControlsSessionManager) GetCurrentSession() (*IGlobalSystemMediaTransportControlsSession, error) {
	var r *IGlobalSystemMediaTransportControlsSession
	hr, _, _ := syscall.Syscall(
		v.VTable().GetCurrentSession,
		2,
		uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(&r)),
		0)
	if hr != 0 {
		return nil, ole.NewError(hr)
	}
	return r, nil
}

type IGlobalSystemMediaTransportControlsSession struct {
	ole.IInspectable
}

type IGlobalSystemMediaTransportControlsSessionVtbl struct {
	ole.IInspectableVtbl
	GetSourceAppUserModelId         uintptr
	TryGetMediaPropertiesAsync      uintptr
	GetTimelineProperties           uintptr
	GetPlaybackInfo                 uintptr
	TryPauseAsync                   uintptr
	TryStopAsync                    uintptr
	TryRecordAsync                  uintptr
	TryFastForwardAsync             uintptr
	TryRewindAsync                  uintptr
	TrySkipNextAsync                uintptr
	TrySkipPreviousAsync            uintptr
	TryChangeChannelUpAsync         uintptr
	TryChangeChannelDownAsync       uintptr
	TryTogglePlayPauseAsync         uintptr
	TryChangeAutoRepeatModeAsync    uintptr
	TryChangePlaybackRateAsync      uintptr
	TryChangeShuffleActiveAsync     uintptr
	TryChangePlaybackPositionAsync  uintptr
	AddTimelinePropertiesChanged    uintptr
	RemoveTimelinePropertiesChanged uintptr
	AddPlaybackInfoChanged          uintptr
	RemovePlaybackInfoChanged       uintptr
	AddMediaPropertiesChanged       uintptr
	RemoveMediaPropertiesChanged    uintptr
}

func (v *IGlobalSystemMediaTransportControlsSession) VTable() *IGlobalSystemMediaTransportControlsSessionVtbl {
	return (*IGlobalSystemMediaTransportControlsSessionVtbl)(unsafe.Pointer(v.RawVTable))
}

func (v *IGlobalSystemMediaTransportControlsSession) TryGetMediaPropertiesAsync() (*IAsyncOperation[IGlobalSystemMediaTransportControlsSessionMediaProperties], error) {
	var r *IAsyncOperation[IGlobalSystemMediaTransportControlsSessionMediaProperties]
	hr, _, _ := syscall.Syscall(
		v.VTable().TryGetMediaPropertiesAsync,
		2,
		uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(&r)),
		0)
	if hr != 0 {
		return nil, ole.NewError(hr)
	}
	return r, nil
}

type IMediaPlaybackType struct {
	ole.IInspectable
}

type IMediaPlaybackTypeVtbl struct {
	ole.IInspectableVtbl
	GetValue uintptr
}

func (v *IMediaPlaybackType) VTable() *IMediaPlaybackTypeVtbl {
	return (*IMediaPlaybackTypeVtbl)(unsafe.Pointer(v.RawVTable))
}

func (v *IMediaPlaybackType) GetValue() (int, error) {
	var r uint32
	hr, _, _ := syscall.Syscall(
		v.VTable().GetValue,
		2,
		uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(&r)),
		0)
	if hr != 0 {
		return 0, ole.NewError(hr)
	}
	return int(r), nil
}

type IGlobalSystemMediaTransportControlsSessionMediaProperties struct {
	ole.IInspectable
}

type IGlobalSystemMediaTransportControlsSessionMediaPropertiesVtbl struct {
	ole.IInspectableVtbl
	GetTitle           uintptr
	GetSubtitle        uintptr
	GetAlbumArtist     uintptr
	GetArtist          uintptr
	GetAlbumTitle      uintptr
	GetTrackNumber     uintptr
	GetGenres          uintptr
	GetAlbumTrackCount uintptr
	GetPlaybackType    uintptr
	GetThumbnail       uintptr
}

func (v *IGlobalSystemMediaTransportControlsSessionMediaProperties) VTable() *IGlobalSystemMediaTransportControlsSessionMediaPropertiesVtbl {
	return (*IGlobalSystemMediaTransportControlsSessionMediaPropertiesVtbl)(unsafe.Pointer(v.RawVTable))
}

func (v *IGlobalSystemMediaTransportControlsSessionMediaProperties) getString(method uintptr) (string, error) {
	var h ole.HString
	hr, _, _ := syscall.Syscall(
		method,
		2,
		uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(&h)),
		0)
	if hr != 0 {
		return "", ole.NewError(hr)
	}
	return h.String(), nil
}

func (v *IGlobalSystemMediaTransportControlsSessionMediaProperties) getInt(method uintptr) (int, error) {
	var r int32
	hr, _, _ := syscall.Syscall(
		method,
		2,
		uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(&r)),
		0)
	if hr != 0 {
		return 0, ole.NewError(hr)
	}
	return int(r), nil
}

func (v *IGlobalSystemMediaTransportControlsSessionMediaProperties) GetTitle() (string, error) {
	return v.getString(v.VTable().GetTitle)
}

func (v *IGlobalSystemMediaTransportControlsSessionMediaProperties) GetSubtitle() (string, error) {
	return v.getString(v.VTable().GetSubtitle)
}

func (v *IGlobalSystemMediaTransportControlsSessionMediaProperties) GetAlbumArtist() (string, error) {
	return v.getString(v.VTable().GetAlbumArtist)
}

func (v *IGlobalSystemMediaTransportControlsSessionMediaProperties) GetArtist() (string, error) {
	return v.getString(v.VTable().GetArtist)
}

func (v *IGlobalSystemMediaTransportControlsSessionMediaProperties) GetAlbumTitle() (string, error) {
	return v.getString(v.VTable().GetAlbumTitle)
}

func (v *IGlobalSystemMediaTransportControlsSessionMediaProperties) GetTrackNumber() (int, error) {
	return v.getInt(v.VTable().GetTrackNumber)
}

func (v *IGlobalSystemMediaTransportControlsSessionMediaProperties) GetGenres() ([]string, error) {
	var r *IVectorView
	hr, _, _ := syscall.Syscall(
		v.VTable().GetGenres,
		2,
		uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(&r)),
		0)
	if hr != 0 {
		return nil, ole.NewError(hr)
	}
	defer r.Release()
	n, err := r.GetSize()
	if err != nil {
		return nil, err
	}
	genres := make([]string, n)
	for i := range genres {
		p, err := r.GetAt(uint32(i))
		if err != nil {
			return nil, err
		}
		s := (*ole.HString)(p).String()
		genres[i] = s
	}
	return genres, nil
}

func (v *IGlobalSystemMediaTransportControlsSessionMediaProperties) GetAlbumTrackCount() (int, error) {
	return v.getInt(v.VTable().GetAlbumTrackCount)
}

func (v *IGlobalSystemMediaTransportControlsSessionMediaProperties) GetPlaybackType() (PlaybackType, error) {
	var r *IMediaPlaybackType
	hr, _, _ := syscall.Syscall(
		v.VTable().GetPlaybackType,
		2,
		uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(&r)),
		0)
	if hr != 0 {
		return PlaybackTypeUnknown, ole.NewError(hr)
	}
	t, err := r.GetValue()
	if err != nil {
		return 0, err
	}
	switch t {
	case 1:
		return PlaybackTypeMusic, nil
	case 2:
		return PlaybackTypeVideo, nil
	case 3:
		return PlaybackTypeImage, nil
	default:
		return PlaybackTypeUnknown, nil
	}
}

var initSync sync.Once
var initErr error

func getInfo(ctx context.Context) (*Info, error) {
	initSync.Do(func() {
		initErr = ole.CoInitializeEx(0, ole.COINIT_MULTITHREADED)
		if initErr != nil {
			runtime.SetFinalizer(&initSync, func(_ interface{}) {
				ole.CoUninitialize()
			})
		}
	})
	if initErr != nil {
		return nil, initErr
	}

	ins, err := ole.RoGetActivationFactory("Windows.Media.Control.GlobalSystemMediaTransportControlsSessionManager", ole.NewGUID("2050C4EE-11A0-57DE-AED7-C97C70338245"))
	if err != nil {
		return nil, err
	}
	managerStatics := (*IGlobalSystemMediaTransportControlsSessionManagerStatics)(unsafe.Pointer(ins))
	managerAsync, err := managerStatics.RequestAsync()
	if err != nil {
		return nil, err
	}
	manager, err := managerAsync.WaitResult("10F0074E-923D-5510-8F4A-DDE37754CA0E")
	if err != nil {
		return nil, err
	}
	session, err := manager.GetCurrentSession()
	if err != nil {
		return nil, err
	}
	propertiesAsync, err := session.TryGetMediaPropertiesAsync()
	if err != nil {
		return nil, err
	}
	properties, err := propertiesAsync.WaitResult("84593A3D-951A-55B6-8353-5205E577797B")
	if err != nil {
		return nil, err
	}
	title, err := properties.GetTitle()
	if err != nil {
		return nil, err
	}
	subtitle, err := properties.GetSubtitle()
	if err != nil {
		return nil, err
	}
	albumArtist, err := properties.GetAlbumArtist()
	if err != nil {
		return nil, err
	}
	artist, err := properties.GetArtist()
	if err != nil {
		return nil, err
	}
	albumTitle, err := properties.GetAlbumTitle()
	if err != nil {
		return nil, err
	}
	trackNumber, err := properties.GetTrackNumber()
	if err != nil {
		return nil, err
	}
	genres, err := properties.GetGenres()
	if err != nil {
		return nil, err
	}
	albumTrackCount, err := properties.GetAlbumTrackCount()
	if err != nil {
		return nil, err
	}
	playbackType, err := properties.GetPlaybackType()
	if err != nil {
		return nil, err
	}
	return &Info{
		AlbumArtists:    stringToArr(albumArtist),
		Artists:         stringToArr(artist),
		Genres:          genres,
		Album:           albumTitle,
		AlbumTrackCount: albumTrackCount,
		Subtitle:        subtitle,
		Title:           title,
		TrackNumber:     trackNumber,
		PlaybackType:    playbackType,
	}, nil
}

func stringToArr(s string) []string {
	if len(s) == 0 {
		return nil
	}
	return []string{s}
}
