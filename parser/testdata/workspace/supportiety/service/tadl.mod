# ...describes a microservice.
# The generate directive will emit a Go project with various dependencies.
module SupportietyMicroservice {

	generate {
		go {
			module = "github.com/worldiety/supportiety"
			output = "{{env `WORKSPACE_DIR`}}/supportiety/service"

			// import defines standard library imports, however may be external anyway, cannot control that.
			// The identifiers must be unique for the entire module.
			import {
			    # ...provides access to atomic primitives.
			    sync "sync"
			}

			// require defines external dependencies
			require {
			    # ...provides CLDR stuff which is not present in the standard library.
			    "golang.org/x/text" @ "v0.3.0" import {
			            mytext "golang.org/x/text"
			            otherpkg "golang.org/x/text/subpackage"
			    }
			}
		}
	}

}