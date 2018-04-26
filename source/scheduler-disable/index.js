'use strict'

// Disable scheduler
// event:
// { "instanceId": "i-00e92a5a9cb7eeb4d" }

const AWS = require('aws-sdk')
const ec2 = new AWS.EC2()

const scheduleTag = process.env.scheduleTag

// handler
exports.handler = (event, context, callback) => {
  console.log(event)

  const instanceId = event.instanceId
  const params = {
    InstanceIds: [
      instanceId
    ]
  }

  ec2.describeInstances(params, (err, instancesData) => {
    if (err) {
      console.log(err, err.stack)
      return callback('ServerError')
    } else {
      if (instancesData.Reservations.length !== 0) {
        const instance = instancesData.Reservations[0].Instances[0]
        const tags = instance.Tags.reduce((tagsObj, tag) => Object.assign(tagsObj, { [tag.Key]: tag.Value }), {})

        // disable range (#)
        if (!tags[scheduleTag].match(/#/)) {
          disableScheduler(instanceId, tags).then((data) => {
            console.log(`${instanceId} scheduler disabled`)
          }).catch((err) => {
            console.log(err, err.stack)
            return callback('ServerError')
          })
        } else {
          console.log(`${instanceId} scheduler already disabled`)
        }
      } else {
        console.log(`instance ${instanceId} doesn't exist`)
        return callback(`instance ${instanceId} doesn't exist`)
      }
    }
  })
}

// disable scheduler
function disableScheduler (instanceId, tags) {
  const range = tags[scheduleTag]
  const params = {
    Resources: [
      instanceId
    ],
    Tags: [
      {
        Key: scheduleTag,
        Value: `#${range}`
      }
    ]
  }

  return ec2.createTags(params).promise()
}

// eof
