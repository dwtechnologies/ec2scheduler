'use strict'

// Set scheduler
// event:
// { "instanceId": "i-00e92a5a9cb7eeb4d", "rangeTime": "07:00-19:00" }
// { "instanceId": "i-00e92a5a9cb7eeb4d", "rangeTime": "07:00-19:00", "rangeWeekdays": "2,3,5" }

const AWS = require('aws-sdk')
const ec2 = new AWS.EC2()

const scheduleTag = process.env.scheduleTag
const scheduleTagDay = process.env.scheduleTagDay

// handler
exports.handler = (event, context, callback) => {
  console.log(event)

  const instanceId = event.instanceId
  const rangeTime = event.rangeTime
  const rangeWeekdays = event.rangeWeekdays

  if (rangeTime.match(/#?\d{2}:\d{2}-\d{2}:\d{2}/)) {
    setScheduler(instanceId, rangeTime, rangeWeekdays).then((data) => {
      console.log(`${instanceId} scheduler set.`)
    }).catch((err) => {
      console.log(err, err.stack)
      return callback('ServerError')
    })
  } else {
    console.log('Invalid time range')
    return callback('ServerError')
  }
}

function setScheduler (instanceId, rangeTime, rangeWeekdays) {
  var params = {}

  if (rangeWeekdays) {
    params = {
      Resources: [
        instanceId
      ],
      Tags: [
        {
          Key: scheduleTagDay,
          Value: rangeWeekdays
        },
        {
          Key: scheduleTag,
          Value: rangeTime
        }
      ]
    }
  } else {
    params = {
      Resources: [
        instanceId
      ],
      Tags: [
        {
          Key: scheduleTag,
          Value: rangeTime
        }
      ]
    }
  }

  return ec2.createTags(params).promise()
}

// eof
