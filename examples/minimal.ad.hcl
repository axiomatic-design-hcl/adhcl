# Minimal example: a thermostat with two root requirements.
# Run: ./bin/adhcl validate examples/minimal.ad.hcl
design "thermostat" {
  boundary = "Keep one room near its set point and return its latest temperature reading. The heater and the sensor are outside the boundary."
}

need "steady_room" {
  statement = "Keep the room at a steady temperature."
  why = "Occupants stay comfortable when the room does not swing up and down."
  origin = "proposed"
}

need "current_reading" {
  statement = "Read the room temperature on request without a long wait."
  origin = "proposed"
}

fr "hold_setpoint" {
  need = [need.steady_room]
  statement = "Keep the room within one degree Celsius of the set point."
  criterion "band" {
    statement = "The room temperature stays within one degree of the set point."
    metric = "absolute difference between the room temperature and the set point"
    op = "<="
    value = 1
    unit = "degC"
    origin = "proposed"
  }
}

fr "report_reading" {
  need = [need.current_reading]
  statement = "Return the latest room temperature reading when asked."
  criterion "reply_time" {
    statement = "The reply arrives within five seconds of the request."
    metric = "seconds from request to reply"
    op = "<="
    value = 5
    unit = "s"
    origin = "proposed"
  }
}

dp "heater_control" {
  controls = [fr.hold_setpoint]
  responsibility = "Switch the heater on when the room is below the set point and off when it is above."
  decision = "selected"
}

dp "reading_store" {
  controls = [fr.report_reading]
  responsibility = "Keep the most recent reading and return it on request."
  decision = "selected"
}

influence {
  from = [dp.heater_control]
  to = fr.hold_setpoint
  effect = "material"
  mechanism = "The switching rule sets how fast the room reaches the set point and how closely it stays there."
}

influence {
  from = [dp.reading_store]
  to = fr.report_reading
  effect = "material"
  mechanism = "The stored value is the reply, so the store decides what the reply contains."
}

influence {
  from = [dp.reading_store]
  to = fr.hold_setpoint
  effect = "none"
  because = "The store holds readings only, so it cannot change the heater switching."
}

cells "other_pairs" {
  scope = design.thermostat
  unlisted = "none"
  because = "No other pair of controls has a material effect. Heater switching does not change the reply time."
}
