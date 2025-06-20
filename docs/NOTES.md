
# Experiment Notes

## Problem
The k3s/k8s load balancing does not seem to function properly. Making `curl`
requests to the endpoint seems to change which pod it hits, but when connecting
via a browser, it always connects to the same pod.

## Thoughts/Intermediate findings
At first, I thought it might have been an added `"Keep-Alive"` header being
added somewhere, but after configuring this, it does not seem like that is the
case After researching, it seems like the `LoadBalancer` type for the service
does not actually mean it strictly load balances the traffic.

Load balancing through `curl` seems to actually load balance, so it seems that
the browser adds additional data that either lets the `KlipperLB` know where to
route it (meaning to the same pod), or some other trickery is happening.

## Solution/Takeaways
`KlipperLB` seems to have this problem. In production/actual use case,
`KlipperLB` might work, as different clients will connect to different pods.
However if I want to make the connections more ephemeral, then I would have to
either introduce an external load balancer (which is a common solution), or I
would have to replace `KlipperLB` with something like `Metallb`.

Additionally it is to be noted that load balancers are defined externally but
used within a cluster, this means that when they are used on a platform like AWS
(`ELB`), you are probably going to get charged for the load balancer service
separately as well as for the k8s setup. Ingresses are used for several
services.

To get back to the point, for testing different pods under load balancing
circumstances, I would have to either use `curl`, or I would have open up a
completely different browser session/window.


## Problem
I need to be able to communicate between pods in the cluster, this means I need
some way of identifying them, most likely with their pod names. In my
application I need to read them. In addition to this, I need to make sure that
the names are somewhat stable, as they currently have hash values appended to
their name which may change if they are restarted for whatever reason.

## Thoughts/Intermediate findings
In a pod, there seems to be two main ways of querying another pod, both
involving the service. 

The other way to query the pod would be to use its endpoint directly, something
like `http://10.42.0.31/8080`. For this, we need to configure a service that
allows requests to this endpoint actually reach them. If we go through a load
balancer service, this would not work.

A node (well the pod in a node) can also be sent a request also at
`http://<pod-id>.<service-name>:<port>`. This requires a headless service, and
for the resource for the pods to specify that this service in a `serviceName`
attribute. This updates the pods' DNS so they can successfully resolve each
other using the URL. 

It's important to note that using non-stateful set replicas is still difficult,
as we have to know the pod id. Using stateful sets can fix this as their pod ids
are ordinal rather than using a hash. For my use case, a Raft-powered HDFS,
stateful sets have the other nice feature of volumes.

In the k8s documentation, under the specification about DNS and A/AAAA records,
it states that `pod-ipv4-address.my-namespace.pod.cluster-domain.example` (aka.
something like `172-17-0-3.default.pod.cluster.local`) is/was a valid way of
referring to a pod through Kube-DNS.

Calling `nslookup`/`dig` can return us the names of the other pods in the
cluster if we have a headless service (where the `clusterIp` is set to `None`).

K8s has the **Downward API**, which enables a pod to learn about information
about itself/~~its cluster~~. As well as providing information like the name and
IP of the pod~~IPs of different pods~~, it can provide physical resource limits
allocated to the pod/pods.

We could have a service per pod, and make the request to each respective
service/pod combo, but this would result in the same issue as earlier: their
discovery.

**Operators/CRDs** [1] seem like a good option and the natural choice for
solving this problem. There exist operators for postgres [2] and mysql so that
there is high availiblity and replication. It seems like they can coordinate
pods, but I'm not sure how integrated this API has to be in order to work.

While operators seem like they could be useful, especially for their
auto-configuration customizability, it does not seem like they would be a good
fit for the consensus/election portion of our application. Operators work to
move a system into a desired state. Our system does not have one desired state.
We might be able to use a meta-operator in this respect, but that is deeper
knowledge than I am currently working at right now. The Postgres operator does
not facilitate instance roles and communication, but instead works to keep each
instance in a predefined state, and the operator itself conducts the
synchronization periodically.

A CRD could help abstract away/organize details about the pods and clusters.
More specifically, if we are to have a specific naming format to make discovery
easier, then using a CRD could hide these details from the user so their
manifest is cleaner.

It seems that there are three ways to solve this problem:
1. Use ephemeral pods through a deployment and configure Raft/the pod itself to
   query the service for the other pods in the cluster/use `nslookup`. Use the result of this
   query to send out the requests.
2. Use stateful sets and a headless service, determine the other pods' addresses
   by passing the number of replicas and name as environment variables. The
   names of these pods will have a consistent naming (ie `app-name-<number>`).
3. Use a custom controller/operator to configure the pod on its startup, passing
   in the required pod IPs for the cluster. On the death of a pod, the operator
   could resend an updated configuration.

Note that strategies 2 and 3 are not mutually exclusive. We'd probably want to
use stateful sets in any scenario. We'd also want to use a headless service even
if we use a custom operator/controller.

For strategy 3, we would want to inform pods when the cluster membership
changes and save logs to disk when upgrading the pod's image. I don't think
these are particularly domain-knowledged based, so the choice would most likely
be a custom controller rather than a custom operator.

## Solution/Takeaways
This took longer than expected.

It is fairly clear that using the **Downward API** to pass the needed cluster
information--which at the very least would be the pod's own identification, in
what ever form that may be.

For the creation of the actual URLs and the passing of messages between
replicas, it seems like the best option would be to use a `StatefulSet` for the
stability of the ip addresses and predictability of the replica identifiers.

I will say, while k8s's emphasis on stateless application has been waining over
the past several years, I don't believe that this is a rare use case and it
should be documented much better. Considering config changes, of the Raft system
itself that is, k8s may still not perfectly fit this use case as on the face of
it, it seems more complicated than anticipated. However, stateful sets will help
with log persistence, which is a nice addition. This was the obvious choice, but
checking to see if there were any alternatives seemed important.

## Citations
[1]: K8s Operator Whitepaper -
    https://github.com/cncf/tag-app-delivery/blob/163962c4b1cd70d085107fc579e3e04c2e14d59c/operator-wg/whitepaper/Operator-WhitePaper_v1-0.md

[2]: Postgres Operator Design -
    https://github.com/zalando/postgres-operator/blob/master/docs/index.md
