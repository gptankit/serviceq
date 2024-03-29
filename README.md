# ServiceQ [![Build Status](https://travis-ci.com/gptankit/serviceq.svg?branch=master)](https://travis-ci.com/gptankit/serviceq) [![GoDoc](https://godoc.org/github.com/gptankit/serviceq?status.svg)](https://pkg.go.dev/github.com/gptankit/serviceq?tab=subdirectories)

ServiceQ is a fault-tolerant gateway for HTTP clusters. It employs probabilistic routing to distribute load during partial cluster shutdown (k/n nodes experiencing downtimes, timeouts, connection loss etc) and queues requests during total cluster shutdown (n nodes down). The queued requests are forwarded in FIFO order when the cluster is available next.

Below graph shows the routing probability (P) on a down node (D) in a 8-node cluster with respect to number of requests (r). Notice how quickly the routing probability on D reduces as the requests on D start to fail. Depending on the rate of request, it will only take a few seconds (sometime even milliseconds) to move all requests away from D, thus ensuring more requests are routed to healthier nodes.

<p>
<img src="https://github.com/gptankit/illustrations/blob/master/serviceq/prob-8.png?raw=true" style="width:50%"/> 
</p>

Note that, even when requests keep failing on D (however less), ServiceQ retries them on other nodes until they succeed. If they do not succeed on any of the nodes, they are queued and periodically retried on the cluster (using the same approach above), until they succeed.

<b>Noticeable features</b>

* HTTP Load Balancing<br/>
* Probabilistic node selection based on error feedback<br/>
* Failed request queueing and deferred forwarding<br/>
* Upfront request queueing<br/>
* Request retries<br/>
* Concurrent connections limit<br/>
* Complete TLS/SSL support (automatic and manual)

Here are the steps to run ServiceQ - </br>

<b>Download</b>

Clone the project into any directory in your workspace <br/>

<pre>
$ git clone https://github.com/gptankit/serviceq
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

This will create a folder <i>serviceq</i> in <i>/usr/local</i> directory and copy the <i>serviceq</i> binary (generated in the build step) to <i>/usr/local/serviceq</i> and <i>sq.properties</i> (serviceq configuration file) to <i>/usr/local/serviceq/config</i>.<br/>

<b>How to Run</b>

Before running, make sure the mandatory configurations in <i>/usr/local/serviceq/config/sq.properties</i> are set (<b>LISTENER_PORT</b>, <b>PROTO</b>, <b>ENDPOINTS</b>, <b>CONCURRENCY_PEAK</b>). The configuration file closely resembles a typical <b>INI</b> file so its fairly easy to understand and make changes -</br>

<pre>
#sq.properties

#Port on which serviceq listens on
LISTENER_PORT=5252

#Protocol the endpoints listens on -- 'http' for both http/https
PROTO=http

#Endpoints seperated by comma (,) -- no spaces allowed, can be a combination of http/https
ENDPOINTS=http://my.server1.com:8080,http://my.server2.com:8080,http://my.server3.com:8080

#Concurrency peak defines how many max concurrent connections are allowed to the cluster
CONCURRENCY_PEAK=2048
</pre>

Also, verify timeout value (default is set to 5s). Low value is preferable as it allows retries to be faster -</br>

<pre>
#Timeout (s) is added to each outgoing request to endpoints, the existing timeouts are overriden, value of -1 means no timeout
OUTGOING_REQUEST_TIMEOUT=5
</pre>

By default deferred queue is enabled with all methods and routes allowed. These options can be controlled as -</br>

<pre>
#Enable deferred queue for requests on final failures (cluster down)
ENABLE_DEFERRED_Q=true

#Request format allows given method/route on deferred queue -- picked up if ENABLE_DEFERRED_Q is true
#Q_REQUEST_FORMATS=POST /orders,PUT,PATCH,DELETE
#Q_REQUEST_FORMATS=ALL
Q_REQUEST_FORMATS=POST,PUT,PATCH,DELETE
</pre>

(Note that Q_REQUEST_FORMATS is also considered if ENABLE_UPFRONT_Q is true)

After all is set - </br>

<pre>$ sudo /usr/local/serviceq/serviceq</pre>

Refer wiki for more details: https://github.com/gptankit/serviceq/wiki

Feel free to play around and post feedbacks
