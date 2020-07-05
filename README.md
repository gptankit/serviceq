<h2>ServiceQ</h2>

ServiceQ is an adaptive gateway for cluster deployments. It employs a probabilistic approach to distribute load and buffers requests during adverse cluster states (downtimes, service unavailability, connection loss etc). The buffered requests are forwarded in FIFO order when the cluster is available next.

Noticeable features -

* HTTP Load Balancing<br/>
* Probabilistic node selection based on error feedback<br/>
* Failed request buffering and deferred forwarding<br/>
* Request retries<br/>
* Concurrent connections limit<br/> 

Here are the steps to run serviceq - </br>

<b>Download</b>

Clone the project into any directory in your workspace (say '<i>serviceq</i>')<br/>

<pre>
$ mkdir serviceq
$ git clone https://github.com/gptankit/serviceq serviceq/
</pre>

Change into directory <i>serviceq</i><br/>

<b>How to Build</b>

<pre>$ make ('make build' will also work)</pre>

Optional: <i>make</i> with debug symbols removed (~25% size reduction)

<pre>$ make build-nodbg</pre>

This will create a Go binary <i>serviceq</i> in the current directory

<b>How to Install</b>

Make sure the current user has root privileges, then - </br>

<pre>$ make install</pre>

This will create a folder <i>serviceq</i> in <i>/usr/local</i> directory and copy the generated <i>serviceq</i> binary to <i>/usr/local/serviceq</i> and <i>sq.properties</i> file (load balancer configuration) to <i>/usr/local/serviceq/config</i>.<br/>

<b>How to Run</b>

Before installing, make sure the mandatory configurations in <i>sq.properties</i> are set (<b>LISTENER_PORT</b>, <b>PROTO</b>, <b>ENDPOINTS</b>, <b>CONCURRENCY_PEAK</b>) -</br>

<pre>
#sq.properties

#Port on which serviceq listens on
LISTENER_PORT=5252

#Protocol the endpoints listens on -- 'http' for both http/https
PROTO=http

#Endpoints seperated by comma (,) -- no spaces allowed, can be a combination of http/https
ENDPOINTS=https://api.server0.com:8000,https://api.server1.com:8001,https://api.server2.com:8002

#Concurrency peak defines how many max concurrent connections are allowed to the cluster
CONCURRENCY_PEAK=2048
</pre>

By default deferred queue is enabled with all methods and routes allowed. These options can be controlled as -</br>

<pre>
#Enable deferred queue for requests on final failures (cluster down)
ENABLE_DEFERRED_Q=true

#Request format allows given method/route on deferred queue -- picked up if ENABLE_DEFERRED_Q is true
#DEFERRED_Q_REQUEST_FORMATS=POST /orders,PUT,PATCH,DELETE
#DEFERRED_Q_REQUEST_FORMATS=ALL
DEFERRED_Q_REQUEST_FORMATS=POST,PUT,PATCH,DELETE
</pre>

After all is set - </br>

<pre>$ sudo /usr/local/serviceq/serviceq</pre>

Feel free to play around and post feedbacks
