'use strict'

// Scheduler
//
// Tag: Schedule
// times are in UTC
// 08:00-19:00   start the instance at 08:00, stop it at 19:00
// 19:00-03:00   start the instance at 19:00, stop it at 03:00 the next day
// #08:00-19:00  ignored
//
// Tag: ScheduleDay
// day of the week in ISO format: 1 Monday, 7 Sunday
// 1,2,3,4,5  runs Mon-Fri (default)
// 2,3,5      run Tue, Wed, Fri

const moment = require('moment')
const format = 'hh:mm'
const AWS = require('aws-sdk')
const ec2 = new AWS.EC2()

const scheduleTag = process.env.scheduleTag
const scheduleTagDay = process.env.scheduleTagDay

// handler
exports.handler = (event, context, callback) => {
  const now = moment() // current time
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

  console.log(`time: ${now}`)
  ec2.describeInstances(params, (err, instancesData) => {
    if (err) {
      console.log(err, err.stack)
      return callback('ServerError')
    } else {
      if (instancesData.Reservations.length !== 0) {
        instancesData.Reservations.forEach((instanceData) => {
          const instance = instanceData.Instances[0]
          const tags = instance.Tags.reduce((tagsObj, tag) => Object.assign(tagsObj, { [tag.Key]: tag.Value }), {})

          const expectedState = shouldRun(instance, now, tags)
          console.log(`${instance.InstanceId} expected state: ${expectedState} (${tags[scheduleTag]})`)

          fixState(instance, expectedState).then((stateData) => {
            if (stateData) {
              console.log(stateData)
            }
          }).catch((err) => {
            console.log(err, err.stack)
            return callback('ServerError')
          })
        })
      }
    }
  })
}

//
// check if day is weekend
// const isWeekend = now => now.isoWeekday() === 6 || now.isoWeekday() === 7
// by default run weekdays (1,2,3,4,5)
const shouldRunDay = (now, rangeWeekdays = [1, 2, 3, 4, 5]) => { return rangeWeekdays.includes(now.weekday()) }

// check if instance should run according to the ScheduleTag
function shouldRun (instance, now, tags) {
  const currentState = instance.State['Code']
  const range = tags[scheduleTag]
  const rangeWeekdays = tags[scheduleTagDay]

  // is disabled (#)
  if (range.match(/#/)) {
    console.log(`${instance.InstanceId} scheduler is disabled (#)`)
    return currentState
  }

  // should not run today
  if (!shouldRunDay(now, rangeWeekdays)) {
    console.log(`${instance.InstanceId} should run today: false`)
    return 80
  }

  // range format
  if (range.match(/\d{2}:\d{2}-\d{2}:\d{2}/)) {
    const startStopTime = range.split('-')
    const startTime = moment(startStopTime[0], format)
    const stopTime = moment(startStopTime[1], format)

    // debugging
    console.log(`${instance.InstanceId} start time: ${startTime}`)
    console.log(`${instance.InstanceId} stop time: ${stopTime}`)

    // startTime-stopTime same day (07:00-19:30)
    if (startTime.isBefore(stopTime)) {
      if (now.isBetween(startTime, stopTime)) {
        return 16 // running
      } else {
        return 80 // stopped
      }

      // startTime-stopTime between days (22:00-03:00 = 22:00-23:59,00:00-03:00)
    } else if (now.isBetween(startTime, moment('23:59', format)) || now.isBetween(now.clone().startOf('day'), stopTime)) {
      return 16 // running
    } else {
      return 80 // stopped
    }
  } else {
    return callback('Wrong range format, expected hh:mm-hh:mm')
  }
}

// fix instance state (running/stopped)
function fixState (instance, expectedState) {
  const currentState = instance.State['Code']
  const params = {
    InstanceIds: [
      instance.InstanceId
    ]
  }

  if (currentState !== expectedState) {
    if (expectedState === 16) {
      return ec2.startInstances(params).promise()
    } else if (expectedState === 80) {
      return ec2.stopInstances(params).promise()
    }
  } else {
    return Promise.resolve(null)
  }
}

// eof
