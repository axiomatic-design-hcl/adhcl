design "inherit" {
  boundary = "A design whose child requirements inherit their need"
}

need "outcome" {
  statement = "The outcome that the user needs"
}

fr "root" {
  need      = need.outcome
  statement = "Give the outcome"
}

dp "root" {
  controls       = [fr.root]
  responsibility = "Root mechanism"
  decision       = "selected"
}

fr "first" {
  parent    = dp.root
  statement = "Give the first part"
}

fr "second" {
  parent    = dp.root
  statement = "Give the second part"
}

dp "first" {
  controls       = [fr.first]
  responsibility = "First mechanism"
  decision       = "selected"
}

dp "second" {
  controls       = [fr.second]
  responsibility = "Second mechanism"
  decision       = "selected"
}

dp "unused" {
  controls       = [fr.second]
  responsibility = "Rejected mechanism"
  decision       = "rejected"
}

influence {
  from             = [dp.first]
  to               = fr.first
  effect           = "material"
  mechanism        = "The first mechanism sets the first part"
  mechanism_source = "inferred"
}

influence {
  from      = [dp.second]
  to        = fr.second
  effect    = "material"
  mechanism = "The second mechanism sets the second part"
}

influence {
  from      = [dp.first]
  to        = fr.second
  effect    = "material"
  mechanism = "The second part reads the first part"
}

influence {
  from    = [dp.second]
  to      = fr.first
  effect  = "none"
  because = "The first part does not read the second part"
}

influence {
  from      = [dp.root]
  to        = fr.root
  effect    = "material"
  mechanism = "The root mechanism gives the outcome"
}
