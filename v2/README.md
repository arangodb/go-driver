# ArangoDB Go Driver V2

Implementation of Driver V2 make use of runtime JSON/VPACK serialization, reducing memory and CPU Driver usage.

## Features

| Implemented | Test Coverage | Description                                |
|-------------|---------------|--------------------------------------------|
|  ✓          |  ✓            | HTTP JSON & VPACK Connection               |
|  -          |  -            | HTTP2 JSON & VPACK Connection              |
|  -          |  -            | VST Connection                             |
|  +          |  -            | Database API Implementation                |
|  +          |  -            | Collection API Implementation              |
|  ✓          |  ✓            | Collection Document Creation               |
|  +          |  -            | Collection Document Update                 |
|  ✓          |  ✓            | Collection Document Read                   |
|  ✓          |  ✓            | Collection Document Delete                 |
|  ✓          |  ✓            | Collection Index                           |
|  ✓          |  ✓            | Query Execution                            |
|  +          |  -            | Transaction Execution                      |
|  -          |  -            | ArangoDB Operations (Views, Users, Graphs) |
