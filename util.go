package bgc

import (
	"fmt"
	"github.com/viant/dsc"
	"github.com/viant/toolbox"
	"golang.org/x/net/context"
	"google.golang.org/api/bigquery/v2"
	"time"
	"math"
)

const maxStatusCheckErrorRetry = 3

func waitForJobCompletion(service *bigquery.Service, context context.Context, projectID string, jobReferenceID string, timeoutMs int) (*bigquery.Job, error) {
	var waitSoFar = 0
	var job *bigquery.Job
	var err error
	var jobStatusErrCheckCount = 0
	for i := 0; ; i++ {
		statusCall := service.Jobs.Get(projectID, jobReferenceID)
		job, err = statusCall.Context(context).Do()
		if err != nil {
			//in case of job status check error, retry 3 times.
			if jobStatusErrCheckCount > maxStatusCheckErrorRetry {
				return job, fmt.Errorf("failed to check status %v", err)
			}
			jobStatusErrCheckCount++
			time.Sleep(time.Duration(math.Pow(float64(jobStatusErrCheckCount), 2)) * time.Second)
			continue
		}

		if res := job.Status.ErrorResult; res != nil {
			info, _ := toolbox.AsIndentJSONText(job)
			return job, fmt.Errorf("%v: %v", job.Status.ErrorResult.Message, info)
		}
		if job.Status.State == doneStatus {
			return job, nil
		}
		time.Sleep(time.Millisecond * time.Duration(tickInterval*(1+i%20)))
		waitSoFar += tickInterval
		if waitSoFar > timeoutMs {
			break
		}
	}
	var JSON string
	if job != nil {
		JSON, _ = toolbox.AsIndentJSONText(job)
	}
	return job, fmt.Errorf("failed to check job status(timeout): %v", JSON)
}

func getServiceAndContext(connection dsc.Connection) (*bigquery.Service, context.Context, error) {
	client, err := asService(connection.Unwrap(servicePointer))
	if err != nil {
		return nil, nil, fmt.Errorf("unable to unwrap biquery client:%v", err)
	}
	ctx, err := asContext(connection.Unwrap(contextPointer))
	if err != nil {
		return nil, nil, fmt.Errorf("unable to unwrap ctx:%v", err)
	}
	return client, *ctx, nil
}

//GetServiceAndContextForManager returns big query service and context for passed in datastore manager.
func GetServiceAndContextForManager(manager dsc.Manager) (*bigquery.Service, context.Context, error) {
	provider := manager.ConnectionProvider()
	connection, err := provider.Get()
	if err != nil {
		return nil, nil, err
	}
	defer connection.Close()
	service, ctx, err := getServiceAndContext(connection)
	if err != nil {
		return nil, nil, err
	}
	return service, ctx, nil
}
