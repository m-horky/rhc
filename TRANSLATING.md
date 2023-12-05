# Translating

## String contexts

To keep the contexts consistent, we can divide the output of rhc into several smaller blocks.
All messages that can appear in the output should be in their appropriate contexts.

### `header`

The header is a first message that gets printed.
It contains information about the action that's being performed or information that's being displayed.

```
Connecting {{ hostname }} to Red Hat.
This might take a few seconds.
```
```
Connection status for {{ hostname }}:
```

### `checklist`

Checklist is the main part of the output.
It consists of multiple lines in the following format:
```
{{ icon }} {{ message }}
```

There are three messages: on subscription-manager, on insights-client and on rhcd daemon.

```
● Connected to Red Hat Subscription Management
● Connected to Red Hat Insights
● The yggdrasil service is inactive
```
```
● This system is already connected to Red Hat Subscription Management
! Cannot connect to Red Hat Insights
● Activated the Remote Host Configuration daemon
```

### `footer`

```
Manager your connected systems: {{ url }}
```

### `table`

A tab-formatted table containing more detailed information about output that was printed above.

It can be table containing errors:
```
TYPE   STEP      ERROR  
ERROR  insights  cannot connect to Red Hat Insights: exit status 1
```

or time durations:
```
STEP       DURATION  
rhsm       1ms       
insights   0s        
yggdrasil  2ms       
```
