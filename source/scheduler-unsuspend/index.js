'use strict'

// Unsuspend
// event:
// { "instanceId": "i-00e92a5a9cb7eeb4d" }

const AWS = require('aws-sdk')
const ec2 = new AWS.EC2()

const scheduleTag = process.env.scheduleTag
const scheduleTagSuspend = process.env.scheduleTagSuspend

// handler
exports.handler = (event, context, callback) => {
  console.log(event)

  const instanceId = event.instanceId
  const params = {
    InstanceIds: [
      instanceId
    ]
  }

  // fetch instace information
  ec2.describeInstances(params, (err, instancesData) => {
    if (err) {
      console.log(err, err.stack)
      return callback("ServerError")
    } else {
      if (instancesData.Reservations.length !== 0) {
        const instance = instancesData.Reservations[0].Instances[0]
        const tags = instance.Tags.reduce((tagsObj, tag) => Object.assign(tagsObj, { [tag.Key]: tag.Value }), {})

        console.log(`scheduler on instance ${instance.InstanceId} will be unsuspended`)

        Promise.all([deleteSuspendTag(instance), enableScheduleTag(instance, tags)]).then((data) => {
          console.log(`unsuspend scheduler for ${instance.InstanceId}`)
          console.log(`scheduler enabled for: ${instance.InstanceId}`)
        }).catch((err) => {
          console.log(err, err.stack)
          return callback('ServerError')
        })
      } else {
        console.log(`no instance found for ${instanceId}`)
        return callback(`no instance found for ${instanceId}`)
      }
    }
  })
}

// delete scheduleTagSuspend tag
function deleteSuspendTag (instance) {
  const instanceId = instance.InstanceId
  const params = {
    Resources: [
      instanceId
    ],
    Tags: [
      {
        Key: scheduleTagSuspend
      }
    ]
  }

  return ec2.deleteTags(params).promise()
}

// uncomment scheduleTag
function enableScheduleTag (instance, tags) {
  const instanceId = instance.InstanceId
  const range = tags[scheduleTag]
  const params = {
    Resources: [
      instanceId
    ],
    Tags: [
      {
        Key: scheduleTag,
        Value: range.replace(/#/g, '')
      }
    ]
  }

  return ec2.createTags(params).promise()
}

// eof
