'use strict'

// Suspend
// event:
// { "instanceId": "i-00e92a5a9cb7eeb4d", "unsuspendDatetime": "20171117" }

const moment = require('moment')
const AWS = require('aws-sdk')
const ec2 = new AWS.EC2()

const scheduleTag = process.env.scheduleTag
const scheduleTagSuspend = process.env.scheduleTagSuspend

// handler
exports.handler = (event, context, callback) => {
  console.log(event)

  const instanceId = event.instanceId
  const unsuspendDatetime = event.unsuspendDatetime
  const params = {
    InstanceIds: [
      instanceId
    ]
  }

  // fetch instace information
  ec2.describeInstances(params, (err, instancesData) => {
    if (err) {
      console.log(err, err.stack)
      return callback('ServerError')
    } else {
      if (instancesData.Reservations.length !== 0) {
        const instance = instancesData.Reservations[0].Instances[0]
        const tags = instance.Tags.reduce((tagsObj, tag) => Object.assign(tagsObj, { [tag.Key]: tag.Value }), {})

        // verify input date
        if (moment(unsuspendDatetime).isValid()) {
          suspendScheduler(instanceId, unsuspendDatetime, tags).then((data) => {
            console.log(`suspend scheduler for ${instanceId} until ${moment(unsuspendDatetime)}`)
          }).catch((err) => {
            console.log(err, err.stack)
            return callback('ServerError')
          })
        } else {
          console.log(`wrong datetime format: ${unsuspendDatetime}`)
          return callback(`wrong datetime format: ${unsuspendDatetime}`)
        }
      } else {
        console.log(`no instance found for ${instanceId}`)
        return callback(`no instance found for ${instanceId}`)
      }
    }
  })
}

// suspend scheduler
function suspendScheduler (instanceId, unsuspendDatetime, tags) {
  const range = tags[scheduleTag]
  const params = {
    Resources: [
      instanceId
    ],
    Tags: [
      {
        Key: scheduleTagSuspend,
        Value: unsuspendDatetime
      },
      {
        Key: scheduleTag,
        Value: `#${range}`
      }
    ]
  }

  return ec2.createTags(params).promise()
}

// eof
