//go:build !ignore_autogenerated

/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by controller-gen. DO NOT EDIT.

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Interpolator) DeepCopyInto(out *Interpolator) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Interpolator.
func (in *Interpolator) DeepCopy() *Interpolator {
	if in == nil {
		return nil
	}
	out := new(Interpolator)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Interpolator) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InterpolatorInputSecrets) DeepCopyInto(out *InterpolatorInputSecrets) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InterpolatorInputSecrets.
func (in *InterpolatorInputSecrets) DeepCopy() *InterpolatorInputSecrets {
	if in == nil {
		return nil
	}
	out := new(InterpolatorInputSecrets)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InterpolatorList) DeepCopyInto(out *InterpolatorList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Interpolator, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InterpolatorList.
func (in *InterpolatorList) DeepCopy() *InterpolatorList {
	if in == nil {
		return nil
	}
	out := new(InterpolatorList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *InterpolatorList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InterpolatorOutputSecrets) DeepCopyInto(out *InterpolatorOutputSecrets) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InterpolatorOutputSecrets.
func (in *InterpolatorOutputSecrets) DeepCopy() *InterpolatorOutputSecrets {
	if in == nil {
		return nil
	}
	out := new(InterpolatorOutputSecrets)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InterpolatorSpec) DeepCopyInto(out *InterpolatorSpec) {
	*out = *in
	if in.OutputSecrets != nil {
		in, out := &in.OutputSecrets, &out.OutputSecrets
		*out = make([]InterpolatorOutputSecrets, len(*in))
		copy(*out, *in)
	}
	if in.InputSecrets != nil {
		in, out := &in.InputSecrets, &out.InputSecrets
		*out = make([]InterpolatorInputSecrets, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InterpolatorSpec.
func (in *InterpolatorSpec) DeepCopy() *InterpolatorSpec {
	if in == nil {
		return nil
	}
	out := new(InterpolatorSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InterpolatorStatus) DeepCopyInto(out *InterpolatorStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]metav1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InterpolatorStatus.
func (in *InterpolatorStatus) DeepCopy() *InterpolatorStatus {
	if in == nil {
		return nil
	}
	out := new(InterpolatorStatus)
	in.DeepCopyInto(out)
	return out
}
