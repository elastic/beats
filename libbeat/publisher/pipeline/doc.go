// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

/*
Package pipeline implements event publishing pipelines used in libbeat. These pipelines
are used to publish events from a source (e.g. an input in Filebeat) all the way to an output
(e.g. Elasticsearch), via processors and a queue. A pipeline encapsulates all these components
and abstracts over some key internal components as well to make event publishing possible.

The main component, of course, is a *pipeline* itself. When a new pipeline is created
a *client* is returned. Users of the pipeline can use this client to publish
events to the pipeline.

When creating a new pipeline, several configuration options may be specified: e.g.
the *processors* that the event must be passed through, the *queue* implementation (in-memory
or spool-to-disk) to use, and the *group of outputs* to which the pipeline should
ultimately send the processed events.

For managing the group of outputs, the pipeline internally creates an *output controller*.
The output controller creates a *consumer* for the queue. It also creates a *retryer*
to retry failed or cancelled batches of events. Finally, it creates *output workers*
that wrap the functionality of publishing events to the configured set of outputs.

All output workers share an internal *work queue*. The output workers dequeue batches of events
from this work queue and send them to the output for publishing. If publishing is unsuccessful
for some reason, either the output worker or the output *cancels* the batch (depending
on where the failure occurred).

Cancelling a batch sends it to the retryer. When the retryer receives a batch of cancelled
events, it enqueues them onto an internal retry queue. Ultimately, the retryer's job is to
dequeue batches from this internal retry queue and enqueue them back onto the work queue that's
shared by output workers. This ensures that the output workers try to publish these batches again.

TODO: write about ACK handling
TODO: write about signaling from output controller -> consumer
TODO: write about signaling from output controller -> retryer
TODO: write about signaling from retryer -> consumer
*/

package pipeline
