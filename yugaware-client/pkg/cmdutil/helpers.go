/*
Copyright © 2021 Yugabyte Support

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

package cmdutil

import (
	"context"
	"fmt"
	"time"

	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/client/customer_tasks"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/models"
)

func WaitForTaskCompletion(ctx context.Context, ywclient *client.YugawareClient, waitTask *models.YBPTask) error {
	params := customer_tasks.NewTasksListParams().
		WithCUUID(ywclient.CustomerUUID())

	for {
		select {
		case <-time.After(1 * time.Second):
			resp, err := ywclient.PlatformAPIs.CustomerTasks.TasksList(params, ywclient.SwaggerAuth)
			if err != nil {
				return err
			}
			for _, task := range resp.GetPayload() {
				if task.ID == waitTask.TaskUUID {
					if task.Status == "Success" {
						return nil
					} else if task.Status != "Running" {
						return fmt.Errorf("task failed: %s", task.Status)
					}

					ywclient.Log.V(1).Info("task not complete", "task", task)
					break
				}
			}
		case <-ctx.Done():
			ywclient.Log.Info("wait cancelled")
			return nil
		}
	}
}
