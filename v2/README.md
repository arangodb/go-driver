# ArangoDB Go Driver V2

Implementation of Driver V2 make use of runtime JSON/VPACK serialization, reducing memory and CPU Driver usage.

## Features

| Implemented | Test Coverage | Description                                                             |
|-------------|---------------|-------------------------------------------------------------------------|
| [x]         | [x]           | HTTP JSON & VPACK Connection                                            |
| [x]         | [ ]           | HTTP2 JSON & VPACK Connection                                           |
| [x]         | [ ]           | VST Connection                                                          |
| [ ]         | [ ]           | Database API Implementation                                             |
| [ ]         | [ ]           | Collection API Implementation                                           |
| [x]         | [x]           | Collection Document Creation                                            |
| [ ]         | [ ]           | Collection Document Update                                              |
| [x]         | [x]           | Collection Document Read                                                |
| [x]         | [x]           | Query Execution                                                         |
| [x]         | [ ]           | Transaction Execution                                                   |
| [ ]         | [ ]           | ArangoDB Operations (Views, Users, Graphs)                              |