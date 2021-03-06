package common

import (
	"encoding/json"
	"reflect"

	fv1 "github.com/yametech/fuxi/pkg/apis/fuxi/v1"
	"golang.org/x/net/context"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/util/retry"
	//"sort"
)

// WorkloadsSlice query resource results
type WorkloadsSlice []*fv1.Workloads

func (w WorkloadsSlice) Len() int      { return len(w) }
func (w WorkloadsSlice) Swap(i, j int) { w[i], w[j] = w[j], w[i] }
func (w WorkloadsSlice) Less(i, j int) bool {
	if w[i].ObjectMeta.CreationTimestamp.Before(&w[j].ObjectMeta.CreationTimestamp) {
		return true
	}
	return false
}

// ResourceQuery query resource interface
type ResourceQuery interface {
	List(namespace, flag string, pos, size int64, selector interface{}) (*unstructured.UnstructuredList, error)
	Get(namespace, name string, subresources ...string) (runtime.Object, error)
	SharedNamespaceList(namespace string, selector interface{}) (*unstructured.UnstructuredList, error)
	Watch(namespace string, resourceVersion string, timeoutSeconds int64, selector interface{}) (<-chan watch.Event, error)
}

// ResourceApply update resource interface
type ResourceApply interface {
	Apply(namespace, name string, obj *unstructured.Unstructured) (*unstructured.Unstructured, bool, error)
	Patch(namespace, name string, patchData map[string]interface{}) (*unstructured.Unstructured, error)
	Delete(namespace, name string) error
}

type WorkloadsResourceVersion interface {
	SetGroupVersionResource(schema.GroupVersionResource)
	GetGroupVersionResource() schema.GroupVersionResource
}

// WorkloadsResourceHandler all needed interface defined
type WorkloadsResourceHandler interface {
	ResourceQuery
	ResourceApply
	WorkloadsResourceVersion
}

// check the default implemented
var _ WorkloadsResourceHandler = &DefaultImplWorkloadsResourceHandler{}

type DefaultImplWorkloadsResourceHandler struct {
	GroupVersionResource schema.GroupVersionResource
}

func (d *DefaultImplWorkloadsResourceHandler) GetGroupVersionResource() schema.GroupVersionResource {
	return d.GroupVersionResource
}
func (d *DefaultImplWorkloadsResourceHandler) SetGroupVersionResource(g schema.GroupVersionResource) {
	d.GroupVersionResource = g
}

func (d *DefaultImplWorkloadsResourceHandler) List(
	namespace,
	flag string,
	pos,
	size int64,
	selector interface{},
) (*unstructured.UnstructuredList, error) {
	var err error
	var items *unstructured.UnstructuredList
	opts := metav1.ListOptions{}

	if selector == nil || selector == "" {
		selector = labels.Everything()
	}
	switch selector.(type) {
	case labels.Selector:
		opts.LabelSelector = selector.(labels.Selector).String()
	case string:
		if selector != "" {
			opts.LabelSelector = selector.(string)
		}
	}

	if flag != "" {
		opts.Continue = flag
	}
	if size > 0 {
		opts.Limit = size + pos
	}
	items, err = SharedK8sClient.
		ClientV2.
		Interface.
		Resource(d.GetGroupVersionResource()).
		Namespace(namespace).
		List(context.Background(), opts)
	if err != nil {
		return nil, err
	}

	return items, nil
}

func (d *DefaultImplWorkloadsResourceHandler) SharedNamespaceList(namespace string, selector interface{}) (*unstructured.UnstructuredList, error) {
	var err error
	var items *unstructured.UnstructuredList
	opts := metav1.ListOptions{}

	if selector == nil || selector == "" {
		selector = labels.Everything()
	}
	switch selector.(type) {
	case labels.Selector:
		opts.LabelSelector = selector.(labels.Selector).String()
	case string:
		if selector != "" {
			opts.LabelSelector = selector.(string)
		}
	}
	gvr := d.GetGroupVersionResource()
	items, err = SharedK8sClient.
		ClientV2.
		Interface.
		Resource(gvr).
		Namespace(namespace).
		List(context.Background(), opts)
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (d *DefaultImplWorkloadsResourceHandler) Get(namespace, name string, subresources ...string) (
	runtime.Object, error,
) {
	gvr := d.GetGroupVersionResource()
	object, err := SharedK8sClient.
		ClientV2.
		Interface.
		Resource(gvr).
		Namespace(namespace).
		Get(context.Background(), name, metav1.GetOptions{}, subresources...)
	if err != nil {
		return nil, err
	}
	return object, nil
}

func (d *DefaultImplWorkloadsResourceHandler) Watch(
	namespace string,
	resourceVersion string,
	timeoutSeconds int64,
	selector interface{},
) (<-chan watch.Event, error) {
	opts := metav1.ListOptions{}
	var err error

	if selector == nil || selector == "" {
		selector = labels.Everything()
	}
	switch selector.(type) {
	case labels.Selector:
		opts.LabelSelector = selector.(labels.Selector).String()
	case string:
		if selector != "" {
			opts.LabelSelector = selector.(string)
		}
	}

	if timeoutSeconds > 0 {
		opts.TimeoutSeconds = &timeoutSeconds
	}

	if resourceVersion != "" {
		opts.ResourceVersion = resourceVersion
	}

	recv, err := SharedK8sClient.
		ClientV2.
		Interface.
		Resource(d.GetGroupVersionResource()).
		Namespace(namespace).
		Watch(context.Background(), opts)
	if err != nil {
		return nil, err
	}

	return recv.ResultChan(), nil
}

func (d *DefaultImplWorkloadsResourceHandler) Apply(namespace string, name string, obj *unstructured.Unstructured) (
	result *unstructured.Unstructured, isUpdate bool, retryErr error) {

	retryErr = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		resource := d.GetGroupVersionResource()
		ctx := context.Background()
		getObj, getErr := SharedK8sClient.
			ClientV2.
			Interface.
			Resource(resource).
			Namespace(namespace).
			Get(ctx, name, metav1.GetOptions{})

		if errors.IsNotFound(getErr) {
			newObj, createErr := SharedK8sClient.
				ClientV2.
				Interface.
				Resource(d.GetGroupVersionResource()).
				Namespace(namespace).
				Create(ctx, obj, metav1.CreateOptions{})
			result = newObj
			return createErr
		}

		if getErr != nil {
			return getErr
		}

		compareObject(getObj, obj)

		newObj, updateErr := SharedK8sClient.
			ClientV2.
			Interface.
			Resource(resource).
			Namespace(namespace).
			Update(ctx, getObj, metav1.UpdateOptions{})

		result = newObj
		isUpdate = true
		return updateErr
	})

	return
}

func (d *DefaultImplWorkloadsResourceHandler) Patch(namespace, name string, pathData map[string]interface{}) (*unstructured.Unstructured, error) {
	ptBytes, err := json.Marshal(pathData)
	if err != nil {
		return nil, err
	}
	gvr := d.GetGroupVersionResource()
	var result *unstructured.Unstructured
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var err error
		result, err = SharedK8sClient.
			ClientV2.
			Interface.
			Resource(gvr).
			Namespace(namespace).
			Patch(context.Background(), name, types.MergePatchType, ptBytes, metav1.PatchOptions{})
		if err != nil {
			return err
		}
		return nil
	},
	)
	return result, retryErr
}

func (d *DefaultImplWorkloadsResourceHandler) Delete(namespace, name string) error {
	gvr := d.GetGroupVersionResource()
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		return SharedK8sClient.
			ClientV2.
			Interface.
			Resource(gvr).
			Namespace(namespace).
			Delete(context.Background(), name, metav1.DeleteOptions{})
	})
	return retryErr
}

func compareMetadataLabelsOrAnnotation(old, new map[string]interface{}) map[string]interface{} {
	newLabels, exist := new["labels"]
	if exist {
		old["labels"] = newLabels
	}
	newAnnotations, exist := new["annotations"]
	if exist {
		old["annotations"] = newAnnotations
	}

	newOwnerReferences, exist := new["ownerReferences"]
	if exist {
		old["ownerReferences"] = newOwnerReferences
	}
	return old
}

func compareObject(getObj, obj *unstructured.Unstructured) {
	if !reflect.DeepEqual(getObj.Object["metadata"], obj.Object["metadata"]) {
		getObj.Object["metadata"] = compareMetadataLabelsOrAnnotation(
			getObj.Object["metadata"].(map[string]interface{}),
			obj.Object["metadata"].(map[string]interface{}),
		)
	}

	if !reflect.DeepEqual(getObj.Object["spec"], obj.Object["spec"]) {
		getObj.Object["spec"] = obj.Object["spec"]
	}

	// configMap
	if !reflect.DeepEqual(getObj.Object["data"], obj.Object["data"]) {
		getObj.Object["data"] = obj.Object["data"]
	}

	if !reflect.DeepEqual(getObj.Object["binaryData"], obj.Object["binaryData"]) {
		getObj.Object["binaryData"] = obj.Object["binaryData"]
	}

	if !reflect.DeepEqual(getObj.Object["stringData"], obj.Object["stringData"]) {
		getObj.Object["stringData"] = obj.Object["stringData"]
	}

	if !reflect.DeepEqual(getObj.Object["type"], obj.Object["type"]) {
		getObj.Object["type"] = obj.Object["type"]
	}

	if !reflect.DeepEqual(getObj.Object["secrets"], obj.Object["secrets"]) {
		getObj.Object["secrets"] = obj.Object["secrets"]
	}

	if !reflect.DeepEqual(getObj.Object["imagePullSecrets"], obj.Object["imagePullSecrets"]) {
		getObj.Object["imagePullSecrets"] = obj.Object["imagePullSecrets"]
	}
	// storageClass field
	if !reflect.DeepEqual(getObj.Object["provisioner"], obj.Object["provisioner"]) {
		getObj.Object["provisioner"] = obj.Object["provisioner"]
	}

	if !reflect.DeepEqual(getObj.Object["parameters"], obj.Object["parameters"]) {
		getObj.Object["parameters"] = obj.Object["parameters"]
	}

	if !reflect.DeepEqual(getObj.Object["reclaimPolicy"], obj.Object["reclaimPolicy"]) {
		getObj.Object["reclaimPolicy"] = obj.Object["reclaimPolicy"]
	}

	if !reflect.DeepEqual(getObj.Object["volumeBindingMode"], obj.Object["volumeBindingMode"]) {
		getObj.Object["volumeBindingMode"] = obj.Object["volumeBindingMode"]
	}
}
