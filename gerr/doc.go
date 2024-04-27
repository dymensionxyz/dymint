// Package gerr provides a systematic way to think about errors. It is based on google API design advice here
// https://cloud.google.com/apis/design/errors
// https://cloud.google.com/apis/design/errors#handling_errors
// https://github.com/googleapis/googleapis/blob/master/google/rpc/code.proto
// In particular, it's important to avoid defining additional errors. Rather, all important errors should wrap one of
// the errors defined in this package.
//
//	"Google APIs must use the canonical error codes defined by google.rpc.Code. Individual APIs must avoid defining
//	additional error codes, since developers are very unlikely to write logic to handle a large number of error codes.
//	For reference, handling an average of three error codes per API call would mean most application logic would just
//	be for error handling, which would not be a good developer experience."
//
// Note: this package can be extended to automatically return the correct GRPC/HTTP codes too, if needed.
// Note: this package could be lifted out and shared across more dymension code, to help us standardise.
package gerr
