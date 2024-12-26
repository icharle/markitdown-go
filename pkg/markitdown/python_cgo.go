package markitdown

/*
#cgo darwin,arm64 CFLAGS: -I${SRCDIR}/../../libs/darwin/include
#cgo darwin,arm64 LDFLAGS: -L${SRCDIR}/../../libs/darwin/lib -lpython3.13

#cgo linux,arm64 CFLAGS: -I${SRCDIR}/../../libs/arm64/include
#cgo linux,arm64 LDFLAGS: -L${SRCDIR}/../../libs/arm64/lib -lpython3.13

#cgo linux,amd64 CFLAGS: -I${SRCDIR}/../../libs/amd64/include
#cgo linux,amd64 LDFLAGS: -L${SRCDIR}/../../libs/amd64/lib -lpython3.13 -lm -ldl -lutil

#include <Python.h>
#include <stdlib.h>
*/
import "C"
import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// PythonManager 用于管理单个 Python 环境
type PythonManager struct {
	markitdownModule *C.PyObject
	markitdownClass  *C.PyObject
}

// managerPool 全局的 PythonManager 池
var managerPool *sync.Pool

// init 初始化 managerPool
func init() {
	managerPool = &sync.Pool{
		New: func() interface{} {
			// 创建一个新的 PythonManager 实例
			pm := &PythonManager{}
			pm.init()
			return pm
		},
	}

	// 注册退出钩子，确保在程序退出时清理所有资源
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
		<-sig
		cleanupAllManagers()
		os.Exit(0)
	}()
}

// init 初始化 Python 环境和加载模块
func (pm *PythonManager) init() {
	// 初始化 Python 解释器
	C.Py_Initialize()

	// 导入 markitdown 模块
	pm.markitdownModule = C.PyImport_ImportModule(C.CString("markitdown"))
	if pm.markitdownModule == nil {
		C.PyErr_Print()
		panic("failed to import markitdown module")
	}

	// 获取 MarkItDown 类
	pm.markitdownClass = C.PyObject_GetAttrString(pm.markitdownModule, C.CString("MarkItDown"))
	if pm.markitdownClass == nil || C.PyCallable_Check(pm.markitdownClass) == 0 {
		C.PyErr_Print()
		panic("failed to get MarkItDown class")
	}
}

// cleanup 清理 Python 资源
func (pm *PythonManager) cleanup() {
	// 释放资源
	if pm.markitdownClass != nil {
		C.Py_DecRef(pm.markitdownClass)
		pm.markitdownClass = nil
	}
	if pm.markitdownModule != nil {
		C.Py_DecRef(pm.markitdownModule)
		pm.markitdownModule = nil
	}

	// Finalize Python
	C.Py_Finalize()
}

// cleanupAllManagers 清理池中的所有 PythonManager 实例
func cleanupAllManagers() {
	for {
		pm := managerPool.Get()
		if pm == nil {
			break
		}
		manager := pm.(*PythonManager)
		manager.cleanup()
	}
}

// getManager 从池中获取一个 PythonManager 实例
func getManager() *PythonManager {
	return managerPool.Get().(*PythonManager)
}

// releaseManager 将 PythonManager 实例放回池中
func releaseManager(pm *PythonManager) {
	managerPool.Put(pm)
}

// callMarkItDown 使用 PythonManager 转换文件
func callMarkItDown(filePath string) (string, error) {
	// 从池中获取 PythonManager
	pm := getManager()
	defer releaseManager(pm) // 调用结束后将实例放回池中

	// 创建 MarkItDown 类的实例
	markitdownInstance := C.PyObject_CallObject(pm.markitdownClass, nil)
	if markitdownInstance == nil {
		C.PyErr_Print()
		return "", fmt.Errorf("failed to create MarkItDown instance")
	}
	defer C.Py_DecRef(markitdownInstance)

	// 获取 convert 方法
	convertMethod := C.PyObject_GetAttrString(markitdownInstance, C.CString("convert"))
	if convertMethod == nil || C.PyCallable_Check(convertMethod) == 0 {
		C.PyErr_Print()
		return "", fmt.Errorf("failed to get convert method")
	}
	defer C.Py_DecRef(convertMethod)

	// 创建参数
	pyFilePath := C.PyUnicode_FromString(C.CString(filePath))
	if pyFilePath == nil {
		C.PyErr_Print()
		return "", fmt.Errorf("failed to create Python string for file path")
	}
	defer C.Py_DecRef(pyFilePath)

	args := C.PyTuple_New(1)
	if args == nil {
		C.PyErr_Print()
		return "", fmt.Errorf("failed to create Python argument tuple")
	}
	defer C.Py_DecRef(args)

	C.PyTuple_SetItem(args, 0, pyFilePath) // PyTuple_SetItem "steals" reference to pyFilePath

	// 调用 convert 方法
	result := C.PyObject_CallObject(convertMethod, args)
	if result == nil {
		C.PyErr_Print()
		return "", fmt.Errorf("failed to execute convert method")
	}
	defer C.Py_DecRef(result)

	// 获取返回值的 text_content 属性
	textContent := C.PyObject_GetAttrString(result, C.CString("text_content"))
	if textContent == nil {
		C.PyErr_Print()
		return "", fmt.Errorf("failed to get text_content attribute")
	}
	defer C.Py_DecRef(textContent)

	// 将返回值转换为 Go 字符串
	goText := C.PyUnicode_AsUTF8(textContent)
	if goText == nil {
		C.PyErr_Print()
		return "", fmt.Errorf("failed to convert text_content to Go string")
	}

	return C.GoString(goText), nil
}
