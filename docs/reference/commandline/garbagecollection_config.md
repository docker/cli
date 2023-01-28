## Description
Garbage collection configuration is an orderd list of prune operations, prune run its operation ones while garbage collection runs
it operation based on its default policy, which we would be looking at in this topic and how to configure them.
Click To know more about [build cache prune.](commandline/builder_prune.md)

Garbage collector works when it is enabled, it runs periodically in the background and follows an ordered list of policies, this 
configuration is done in the `etc/docker/daemon.json` config file, its not a command line configuration.

## Enable GC

The following enables the default GC on your docker daemon: 

```bash
"builder": {
    "gc": {
      "enabled": true
    }
  }
```
The garbage collection config file can be created in case you dont have one located in your `etc/docker` folder, and if you do have
one it migth be there and be set to `false` so you need to enable the garbage collection by setting it as true, if you don't see
any gc JSON, copy the above JSON and paste it in the file. That is the very first step to configure your garbage collection.
Garbage collection config JSON are optional, i.e some fields can be left out, like the example above about enabling GC, the policy
field was left out including the defaultKeepStorage. Next we are going to talk about defaultKeepStorage in the JSON file and
it's configuration.

## defaultKeepStorage
GC works based on  `defaultKeepStorage` and the default keep storage is 10% of local disk space which can be configured,
the `defaultKeepStorage` is very useful to measure the amount of cache to be deleted. When the `defaultKeepStorage` is set
it uses the `defaultKeepStorage` value to set a limit to the amount of build cache to be deleted.
If a defaultKeepStorage is set to 10GB and you have a build cache of 20GB the prune policy deletes all the build cache until it
gets to the defaultKeepStorage value of 10GB.

The following is an example of a defaultKeepStorage GC:

```bash
"builder": {
    "gc": {
      "enabled": true,
      "defaultKeepStorage": "10GB"
    }
  }
```
The example above shows a GC configuration being enabled and the defaultKeepStorage being set to 10GB.

## Garbage Collection Policy 
Gabage collection works in an orderd list of prune operations, They are basically four default policy rules which can be configured
and also be added to or reduced to your preferred choice based on what can handle the build cache. Every policy as we said is a
prune operation with different categories of prune operation, which delete build cache on every stage based on different
conditions.
The reason there are multiple rules, is because the we users needs to be smart about which objects will be used more often in 
future builds.

Below is an illustration of how GC default policy works

```bash
Step 1 ==>   GC enabled
Step 2 ==>   Policy rules :
                           Rule 0 ==> The first policy rule states that: if the build cache uses more than 512MB, delete the most 
                                      easily reproducible data after it has not been used for 2 days.
                                      If the first rule is not enough to bring the cache down, it move to the next policy.
                           
                           Rule 1 ==> This rule only execute if the first rule is not enough to bring the cache down,
                                      this default rule state that: remove any data not used for 60 days, if this rule
                                      is not enough to bring the cache down, it jumps to the third rule, hence it
                                      terminates.
                           
                           Rule 2 ==> The third rule only exexute if the first two rules are not enough to
                                      bring down the build cache. This rule states that: it should keep 
                                      every remaining unshared build cache under cap i.e among the remaining
                                      build cache, remove the ones that are unshared. If this rule is not 
                                      enough to bring the cache down, it jumps to the fourth rule, hence it terminates.
                           
                           Rule 3 ==> This rule only execute if the previous three rules are not enough to bring the down the build
                                      cache. This rule states that: Start deleting internal data to keep build cache under cap. i.e
                                      delete datas to make sure that the build cache is under the keep storage limit.
Step 3 ==>   Keep Storage : 512MB
```
**Note**
>All garbage collection properties are optional, so you can configure the gc to have upto 4 rules or not, also add filters.
>Some of the properties can be left out, maybe step 3 etc

Below is the JSON configuration of the GC default policy:

```bash
"gc": {
      "enabled": true,
      "policy": [
            {"keepStorage": "512MB", "filter": ["unused-for=48h"]},
            {"keepStorage": "512MB", "filter": {"unused-for": {"1440h": true}}}, # days converted to hours
            {"keepStorage": "512MB"},
            {"keepStorage": "512MB", "all": true}
        ]
      "defaultKeepStorage": "512MB"
    }
```

## more example of a random GC Config

Now we are going to look at a new garbage collection config that is not the default policy.

```bash
"gc": {
      "enabled": true,
      "policy": [
            {"keepStorage": "10GB", "filter": ["unused-for=2200h"]},
            {"keepStorage": "50GB", "filter": {"unused-for": {"3300h": true}}},
            {"keepStorage": "100GB", "all": true}
        ]
    }
```
The configuration above shows that the garbage collection is on, and it follows three rules.
First rule state that, if the build cache is more than 10GB delete every unused build cache that are more than 92 days old 
(converted to days), if the first rule is not enough to bring the cache down to 10GB it jumps to the next rule, stating that it 
should remove every cache that are more than 136 days old, if the second rule is not enough to bring the cache down to 50GB, then
it would apply the third rule that state that, it should remove all the build cache data until it the keep storage reaches 100GB.
For every state once the condition is meant, it will terminate and not move to the other condition.

Go on to configure your gabage collection for a better build.
