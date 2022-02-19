<h1>CUE Instance API reference</h1>
<p>Packages:</p>
<ul class="simple">
<li>
<a href="#cue.contrib.flux.io%2fv1alpha1">cue.contrib.flux.io/v1alpha1</a>
</li>
</ul>
<h2 id="cue.contrib.flux.io/v1alpha1">cue.contrib.flux.io/v1alpha1</h2>
<p>Package v1alpha1 contains API Schema definitions for the cue v1alpha1 API group</p>
Resource Types:
<ul class="simple"></ul>
<h3 id="cue.contrib.flux.io/v1alpha1.CrossNamespaceSourceReference">CrossNamespaceSourceReference
</h3>
<p>
(<em>Appears on:</em>
<a href="#cue.contrib.flux.io/v1alpha1.CueInstanceSpec">CueInstanceSpec</a>)
</p>
<div class="md-typeset__scrollwrap">
<div class="md-typeset__table">
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code><br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>API version of the referent.</p>
</td>
</tr>
<tr>
<td>
<code>kind</code><br>
<em>
string
</em>
</td>
<td>
<p>Kind of the referent.</p>
</td>
</tr>
<tr>
<td>
<code>name</code><br>
<em>
string
</em>
</td>
<td>
<p>Name of the referent.</p>
</td>
</tr>
<tr>
<td>
<code>namespace</code><br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Namespace of the referent, defaults to the namespace of the Kubernetes resource object that contains the reference.</p>
</td>
</tr>
</tbody>
</table>
</div>
</div>
<h3 id="cue.contrib.flux.io/v1alpha1.CueInstance">CueInstance
</h3>
<p>CueInstance is the Schema for the cueinstances API</p>
<div class="md-typeset__scrollwrap">
<div class="md-typeset__table">
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>metadata</code><br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code><br>
<em>
<a href="#cue.contrib.flux.io/v1alpha1.CueInstanceSpec">
CueInstanceSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>interval</code><br>
<em>
<a href="https://godoc.org/k8s.io/apimachinery/pkg/apis/meta/v1#Duration">
Kubernetes meta/v1.Duration
</a>
</em>
</td>
<td>
<p>The interval at which the instance will be reconciled.</p>
</td>
</tr>
<tr>
<td>
<code>sourceRef</code><br>
<em>
<a href="#cue.contrib.flux.io/v1alpha1.CrossNamespaceSourceReference">
CrossNamespaceSourceReference
</a>
</em>
</td>
<td>
<p>A reference to a Flux Source from which an artifact will be downloaded
and the CUE instance built.</p>
</td>
</tr>
<tr>
<td>
<code>root</code><br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The module root of the CUE instance.</p>
</td>
</tr>
<tr>
<td>
<code>path</code><br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The path at which the CUE instance will be built from.</p>
</td>
</tr>
<tr>
<td>
<code>package</code><br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The CUE package to use for the CUE instance. This is useful when applying
a CUE schema to plain yaml files.</p>
</td>
</tr>
<tr>
<td>
<code>tags</code><br>
<em>
<a href="#cue.contrib.flux.io/v1alpha1.TagVar">
[]TagVar
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Tags that will be injected into the CUE instance.</p>
</td>
</tr>
<tr>
<td>
<code>tagVars</code><br>
<em>
<a href="#cue.contrib.flux.io/v1alpha1.TagVar">
[]TagVar
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>TagVars that will be available to the CUE instance.</p>
</td>
</tr>
<tr>
<td>
<code>expressions</code><br>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The CUE expression(s) to execute.</p>
</td>
</tr>
<tr>
<td>
<code>dependsOn</code><br>
<em>
<a href="https://godoc.org/github.com/fluxcd/pkg/runtime/dependency#CrossNamespaceDependencyReference">
[]Runtime dependency.CrossNamespaceDependencyReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Dependencies that must be ready before the CUE instance is reconciled.</p>
</td>
</tr>
<tr>
<td>
<code>prune</code><br>
<em>
bool
</em>
</td>
<td>
<p>Prune enables garbage collection.</p>
</td>
</tr>
<tr>
<td>
<code>retryInterval</code><br>
<em>
<a href="https://godoc.org/k8s.io/apimachinery/pkg/apis/meta/v1#Duration">
Kubernetes meta/v1.Duration
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The interval at which to retry a previously failed reconciliation.
When not specified, the controller uses the CueInstanceSpec.Interval
value to retry failures.</p>
</td>
</tr>
<tr>
<td>
<code>timeout</code><br>
<em>
<a href="https://godoc.org/k8s.io/apimachinery/pkg/apis/meta/v1#Duration">
Kubernetes meta/v1.Duration
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Timeout for validation, apply and health checking operations.
Defaults to &lsquo;Interval&rsquo; duration.</p>
</td>
</tr>
<tr>
<td>
<code>suspend</code><br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>This flag tells the controller to suspend subsequent cue executions,
it does not apply to already started executions. Defaults to false.</p>
</td>
</tr>
<tr>
<td>
<code>serviceAccountName</code><br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The name of the Kubernetes service account to impersonate
when reconciling this CueInstance.</p>
</td>
</tr>
<tr>
<td>
<code>kubeConfig</code><br>
<em>
<a href="#cue.contrib.flux.io/v1alpha1.KubeConfig">
KubeConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The KubeConfig for reconciling the CueInstance on a remote cluster.
When specified, KubeConfig takes precedence over ServiceAccountName.</p>
</td>
</tr>
<tr>
<td>
<code>force</code><br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Force instructs the controller to recreate resources
when patching fails due to an immutable field change.</p>
</td>
</tr>
<tr>
<td>
<code>validate</code><br>
<em>
<a href="#cue.contrib.flux.io/v1alpha1.Validation">
Validation
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>TODO(maybe): this could be an array of validations
in which case the policy may need to apply to all resources
would allow for greater flexibility</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code><br>
<em>
<a href="#cue.contrib.flux.io/v1alpha1.CueInstanceStatus">
CueInstanceStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
</div>
</div>
<h3 id="cue.contrib.flux.io/v1alpha1.CueInstanceSpec">CueInstanceSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#cue.contrib.flux.io/v1alpha1.CueInstance">CueInstance</a>)
</p>
<p>CueInstanceSpec defines the desired state of CueInstance</p>
<div class="md-typeset__scrollwrap">
<div class="md-typeset__table">
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>interval</code><br>
<em>
<a href="https://godoc.org/k8s.io/apimachinery/pkg/apis/meta/v1#Duration">
Kubernetes meta/v1.Duration
</a>
</em>
</td>
<td>
<p>The interval at which the instance will be reconciled.</p>
</td>
</tr>
<tr>
<td>
<code>sourceRef</code><br>
<em>
<a href="#cue.contrib.flux.io/v1alpha1.CrossNamespaceSourceReference">
CrossNamespaceSourceReference
</a>
</em>
</td>
<td>
<p>A reference to a Flux Source from which an artifact will be downloaded
and the CUE instance built.</p>
</td>
</tr>
<tr>
<td>
<code>root</code><br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The module root of the CUE instance.</p>
</td>
</tr>
<tr>
<td>
<code>path</code><br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The path at which the CUE instance will be built from.</p>
</td>
</tr>
<tr>
<td>
<code>package</code><br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The CUE package to use for the CUE instance. This is useful when applying
a CUE schema to plain yaml files.</p>
</td>
</tr>
<tr>
<td>
<code>tags</code><br>
<em>
<a href="#cue.contrib.flux.io/v1alpha1.TagVar">
[]TagVar
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Tags that will be injected into the CUE instance.</p>
</td>
</tr>
<tr>
<td>
<code>tagVars</code><br>
<em>
<a href="#cue.contrib.flux.io/v1alpha1.TagVar">
[]TagVar
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>TagVars that will be available to the CUE instance.</p>
</td>
</tr>
<tr>
<td>
<code>expressions</code><br>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The CUE expression(s) to execute.</p>
</td>
</tr>
<tr>
<td>
<code>dependsOn</code><br>
<em>
<a href="https://godoc.org/github.com/fluxcd/pkg/runtime/dependency#CrossNamespaceDependencyReference">
[]Runtime dependency.CrossNamespaceDependencyReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Dependencies that must be ready before the CUE instance is reconciled.</p>
</td>
</tr>
<tr>
<td>
<code>prune</code><br>
<em>
bool
</em>
</td>
<td>
<p>Prune enables garbage collection.</p>
</td>
</tr>
<tr>
<td>
<code>retryInterval</code><br>
<em>
<a href="https://godoc.org/k8s.io/apimachinery/pkg/apis/meta/v1#Duration">
Kubernetes meta/v1.Duration
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The interval at which to retry a previously failed reconciliation.
When not specified, the controller uses the CueInstanceSpec.Interval
value to retry failures.</p>
</td>
</tr>
<tr>
<td>
<code>timeout</code><br>
<em>
<a href="https://godoc.org/k8s.io/apimachinery/pkg/apis/meta/v1#Duration">
Kubernetes meta/v1.Duration
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Timeout for validation, apply and health checking operations.
Defaults to &lsquo;Interval&rsquo; duration.</p>
</td>
</tr>
<tr>
<td>
<code>suspend</code><br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>This flag tells the controller to suspend subsequent cue executions,
it does not apply to already started executions. Defaults to false.</p>
</td>
</tr>
<tr>
<td>
<code>serviceAccountName</code><br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The name of the Kubernetes service account to impersonate
when reconciling this CueInstance.</p>
</td>
</tr>
<tr>
<td>
<code>kubeConfig</code><br>
<em>
<a href="#cue.contrib.flux.io/v1alpha1.KubeConfig">
KubeConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The KubeConfig for reconciling the CueInstance on a remote cluster.
When specified, KubeConfig takes precedence over ServiceAccountName.</p>
</td>
</tr>
<tr>
<td>
<code>force</code><br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Force instructs the controller to recreate resources
when patching fails due to an immutable field change.</p>
</td>
</tr>
<tr>
<td>
<code>validate</code><br>
<em>
<a href="#cue.contrib.flux.io/v1alpha1.Validation">
Validation
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>TODO(maybe): this could be an array of validations
in which case the policy may need to apply to all resources
would allow for greater flexibility</p>
</td>
</tr>
</tbody>
</table>
</div>
</div>
<h3 id="cue.contrib.flux.io/v1alpha1.CueInstanceStatus">CueInstanceStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#cue.contrib.flux.io/v1alpha1.CueInstance">CueInstance</a>)
</p>
<p>CueInstanceStatus defines the observed state of CueInstance</p>
<div class="md-typeset__scrollwrap">
<div class="md-typeset__table">
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ReconcileRequestStatus</code><br>
<em>
<a href="https://godoc.org/github.com/fluxcd/pkg/apis/meta#ReconcileRequestStatus">
github.com/fluxcd/pkg/apis/meta.ReconcileRequestStatus
</a>
</em>
</td>
<td>
<p>
(Members of <code>ReconcileRequestStatus</code> are embedded into this type.)
</p>
</td>
</tr>
<tr>
<td>
<code>observedGeneration</code><br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>ObservedGeneration is the last reconciled generation.</p>
</td>
</tr>
<tr>
<td>
<code>conditions</code><br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#condition-v1-meta">
[]Kubernetes meta/v1.Condition
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>lastAppliedRevision</code><br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The last successfully applied revision.
The revision format for Git sources is <branch|tag>/<commit-sha>.</p>
</td>
</tr>
<tr>
<td>
<code>lastAttemptedRevision</code><br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>LastAttemptedRevision is the revision of the last reconciliation attempt.</p>
</td>
</tr>
<tr>
<td>
<code>inventory</code><br>
<em>
<a href="#cue.contrib.flux.io/v1alpha1.ResourceInventory">
ResourceInventory
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Inventory contains the list of Kubernetes resource object references that have been successfully applied.</p>
</td>
</tr>
</tbody>
</table>
</div>
</div>
<h3 id="cue.contrib.flux.io/v1alpha1.KubeConfig">KubeConfig
</h3>
<p>
(<em>Appears on:</em>
<a href="#cue.contrib.flux.io/v1alpha1.CueInstanceSpec">CueInstanceSpec</a>)
</p>
<p>KubeConfig references a Kubernetes secret that contains a kubeconfig file.</p>
<div class="md-typeset__scrollwrap">
<div class="md-typeset__table">
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>secretRef</code><br>
<em>
<a href="https://godoc.org/github.com/fluxcd/pkg/apis/meta#LocalObjectReference">
github.com/fluxcd/pkg/apis/meta.LocalObjectReference
</a>
</em>
</td>
<td>
<p>SecretRef holds the name to a secret that contains a &lsquo;value&rsquo; key with
the kubeconfig file as the value. It must be in the same namespace as
the CueInstance.
It is recommended that the kubeconfig is self-contained, and the secret
is regularly updated if credentials such as a cloud-access-token expire.
Cloud specific <code>cmd-path</code> auth helpers will not function without adding
binaries and credentials to the Pod that is responsible for reconciling
the CueInstance.</p>
</td>
</tr>
</tbody>
</table>
</div>
</div>
<h3 id="cue.contrib.flux.io/v1alpha1.ResourceInventory">ResourceInventory
</h3>
<p>
(<em>Appears on:</em>
<a href="#cue.contrib.flux.io/v1alpha1.CueInstanceStatus">CueInstanceStatus</a>)
</p>
<p>ResourceInventory contains a list of Kubernetes resource object references that have been applied by a Kustomization.</p>
<div class="md-typeset__scrollwrap">
<div class="md-typeset__table">
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>entries</code><br>
<em>
<a href="#cue.contrib.flux.io/v1alpha1.ResourceRef">
[]ResourceRef
</a>
</em>
</td>
<td>
<p>Entries of Kubernetes resource object references.</p>
</td>
</tr>
</tbody>
</table>
</div>
</div>
<h3 id="cue.contrib.flux.io/v1alpha1.ResourceRef">ResourceRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#cue.contrib.flux.io/v1alpha1.ResourceInventory">ResourceInventory</a>)
</p>
<p>ResourceRef contains the information necessary to locate a resource within a cluster.</p>
<div class="md-typeset__scrollwrap">
<div class="md-typeset__table">
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>id</code><br>
<em>
string
</em>
</td>
<td>
<p>ID is the string representation of the Kubernetes resource object&rsquo;s metadata,
in the format &lsquo;<namespace><em><name></em><group>_<kind>&rsquo;.</p>
</td>
</tr>
<tr>
<td>
<code>v</code><br>
<em>
string
</em>
</td>
<td>
<p>Version is the API version of the Kubernetes resource object&rsquo;s kind.</p>
</td>
</tr>
</tbody>
</table>
</div>
</div>
<h3 id="cue.contrib.flux.io/v1alpha1.TagVar">TagVar
</h3>
<p>
(<em>Appears on:</em>
<a href="#cue.contrib.flux.io/v1alpha1.CueInstanceSpec">CueInstanceSpec</a>)
</p>
<p>TagVar is a tag variable with a required name and optional value</p>
<div class="md-typeset__scrollwrap">
<div class="md-typeset__table">
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code><br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>value</code><br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
</div>
</div>
<h3 id="cue.contrib.flux.io/v1alpha1.Validation">Validation
</h3>
<p>
(<em>Appears on:</em>
<a href="#cue.contrib.flux.io/v1alpha1.CueInstanceSpec">CueInstanceSpec</a>)
</p>
<div class="md-typeset__scrollwrap">
<div class="md-typeset__table">
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>mode</code><br>
<em>
<a href="#cue.contrib.flux.io/v1alpha1.ValidationMode">
ValidationMode
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>schema</code><br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>type</code><br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
</div>
</div>
<h3 id="cue.contrib.flux.io/v1alpha1.ValidationMode">ValidationMode
(<code>string</code> alias)</h3>
<p>
(<em>Appears on:</em>
<a href="#cue.contrib.flux.io/v1alpha1.Validation">Validation</a>)
</p>
<div class="admonition note">
<p class="last">This page was automatically generated with <code>gen-crd-api-reference-docs</code></p>
</div>
