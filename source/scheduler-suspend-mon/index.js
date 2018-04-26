'use strict'

// Suspend Mon

const moment = require('moment')
const AWS = require('aws-sdk')
const ec2 = new AWS.EC2()

const scheduleTag = process.env.scheduleTag
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
          scheduleTagSuspend
        ]
      }
    ]
  }

  // fetch instaces information
  ec2.describeInstances(params, (err, instancesData) => {
    if (err) {
      console.log(err, err.stack)
      return callback('ServerError')
    } else {
      if (instancesData.Reservations.length !== 0) {
        instancesData.Reservations.forEach((instanceData) => {
          const instance = instanceData.Instances[0]
          const tags = instance.Tags.reduce((tagsObj, tag) => Object.assign(tagsObj, { [tag.Key]: tag.Value }), {})

          if (moment().isAfter(moment(tags[scheduleTagSuspend]))) {
            console.log(`scheduler on instance ${instance.InstanceId} will be unsuspended`)

            Promise.all([deleteSuspendTag(instance), enableScheduleTag(instance, tags)]).then((data) => {
              console.log(`unsuspend scheduler for ${instance.InstanceId}`)
              console.log(`scheduler enabled for: ${instance.InstanceId}`)
            }).catch((err) => {
              console.log(err, err.stack)
              return callback('ServerError')
            })
          }
        })
      } else {
        console.log('no suspended instances')
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
