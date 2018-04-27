# ec2scheduler
Set of Lambda functions to manage the scheduling of EC2 instances.


#### Tags

Tags drive the scheduler's logic. Names are configurable from the SAM Template
parameters and are exposed to the functions as environment variables.

- **Schedule**: required for the scheduler engine to work

```
  times are in UTC
  08:00-19:00   start the instance at 08:00, stop it at 19:00
  19:00-03:00   start the instance at 19:00, stop it at 03:00 the next day
  #08:00-19:00  ignored
```

- **ScheduleDay**: optional, defines to which day the scheduler applies

```
  day(s) of the week in ISO format: 1 Monday, 7 Sunday
  1,2,3,4,5  runs Mon-Fri (default)
  2,3,5      run Tue, Wed, Fri
```


- **ScheduleSuspendUntil**: handle by the ec2schedulerSuspend/Unsuspend/SuspendMon functions

```
  2020                  suspend the scheduler until Jan 1 2020 00:00:00
  20180816T13:00        suspend the scheduler until Aug 16 2018 13:00:00
```



#### Functions

- ##### ec2scheduler - (source/scheduler)


Scheduler engine, runs every 5 minutes to verify tagged EC2 instances (**Schedule** tag) should be running (16) or stopped (status 80).





- ##### ec2schedulerSet - (source/scheduler-set)

Set the scheduler for instanceId (create tag if doesn't exists, modify if it exists). Event format:

```json
{
	"instanceId": "i-00e92a5a9cb7eeb4d",
	"rangeTime": "07:00-19:00"
}

{
    "instanceId": "i-00e92a5a9cb7eeb4d",
    "rangeTime": "07:00-19:00",
    "rangeWeekdays": "2,3,5"
}
```





- ##### ec2schedulerDisable - (source/scheduler-disable)

Disable scheduler for instanceId. Event format:

```json
{
    "instanceId": "i-00e92a5a9cb7eeb4d"
}
```





- ##### ec2schedulerStatus - (source/scheduler-status)


Returns a list of instanceIds and their scheduler settings. Output:

```json
{
    "i-00e92a5a9cb7eeb4d":{
        "Schedule":"07:00-10:00",
        "ScheduleDay":"1,3,5"
    }
}
```




- ##### ec2schedulerSuspend - (source/scheduler-suspend)

Suspend a scheduler until **ScheduleSuspendUntil** tag. Adds **ScheduleSuspendUntil** tag and comment out **Schedule** tag. Event format:

```json
{
	"instanceId": "i-00e92a5a9cb7eeb4d",
	"unsuspendDatetime": "20171117"
}
```





- ##### ec2schedulerUnsuspend - (source/scheduler-unsuspend)

Unsuspend a scheduler. Delete **ScheduleSuspendUntil** tag and uncomment **Schedule** tag. Event format:

```json
{
	"instanceId": "i-00e92a5a9cb7eeb4d"
}
```





- ##### ec2schedulerSuspendMon - (source/scheduler-suspend-mon)

Scheduled function that monitors the **ScheduleSuspendUntil** tag. In case the suspend time is expired, the scheduler is unsuspended.

