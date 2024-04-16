# Project Title

Building a pull through cache server for K8s cluster using Golang

## Description

When you have multiple containers running in a cluster that pull images from an external registry, the inter network calls cause a higher latency in pulling images and also accrue higher network costs. Additionally, with external registries, security of the images cannot be guaranteed and scans of any sort cannot be done. 
Having an in house registry solves all the above problems. We don't have to have all images stored in the registry in advance, a pull through cache that acts as a registry mirror does the job of mapping in-house registries to external registries and pulling images that dont exist already. 

## Getting Started

### Dependencies

* k8s cluster
* custom/cloud private registry
* permissions for k8s workload to operate on the registry



### Executing program

* 
```

## Help

Any advise for common problems or issues.
```
command to run if program contains helper info
```

## Authors

Contributors names and contact info

Priyanka Bettadapura

## Version History

* 0.1
    * Initial Release

## License

This project is licensed under the [NAME HERE] License - see the LICENSE.md file for details

## Acknowledgments

Inspiration, code snippets, etc.
* [awesome-readme](https://github.com/matiassingers/awesome-readme)
* [PurpleBooth](https://gist.github.com/PurpleBooth/109311bb0361f32d87a2)
* [dbader](https://github.com/dbader/readme-template)
* [zenorocha](https://gist.github.com/zenorocha/4526327)
* [fvcproductions](https://gist.github.com/fvcproductions/1bfc2d4aecb01a834b46)