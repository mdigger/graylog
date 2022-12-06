// Package graylog provides support for logging to the Graylog server.
//
// It can send messages to the Graylog server using UDP or TCP.
// When using UDP as a transport layer, the messages sent are gzip compressed
// and automatically chunked.
package graylog
