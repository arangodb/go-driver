version=`git describe --tags --long`

cat << EOF > ./version-driver.go
package driver

//go:generate bash ./scripts/update-version.sh
var driverVersion = "$version"
EOF
