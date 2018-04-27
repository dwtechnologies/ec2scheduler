'use strict'

// Scheduler status

const AWS = require('aws-sdk')
const ec2 = new AWS.EC2()

const scheduleTag = process.env.scheduleTag
const scheduleTagDay = process.env.scheduleTagDay
const scheduleTagSuspend = process.env.scheduleTagSuspend

// handler
exports.handler = (event, context, callback) => {
  const params = {
    Filters: [
      {
        Name: 'instance-state-name',
        Values: [
          'running',
          'stopped'
        ]
      },
      {
        Name: 'tag-key',
        Values: [
          scheduleTag
        ]
      }
    ]
  }

  var scheduleStatus = {}
  ec2.describeInstances(params, (err, instancesData) => {
    if (err) {
      console.log(err, err.stack)
      return callback('ServerError')
    } else {
      if (instancesData.Reservations.length !== 0) {
        instancesData.Reservations.forEach((instanceData) => {
          const instance = instanceData.Instances[0]
          const tags = instance.Tags.reduce((tagsObj, tag) => Object.assign(tagsObj, { [tag.Key]: tag.Value }), {})

          scheduleStatus[instance.InstanceId] = {
            Name: tags.Name,
            Status: instance.State.Code,
            [scheduleTag]: tags[scheduleTag],
            [scheduleTagDay]: tags[scheduleTagDay],
            [scheduleTagSuspend]: tags[scheduleTagSuspend]
          }
        })

        console.log(JSON.stringify(scheduleStatus, null, 2))
        // API GW response
        return callback(null, {
          statusCode: 200,
          body: JSON.stringify(scheduleStatus)
        })
      }
    }
  })
}

// eof
