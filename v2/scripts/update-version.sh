version=`git describe --tags --long`

cat << EOF > ../utils/version-driver.go
package utils

//go:generate bash ../scripts/update-version.sh
var driverVersion = "$version"
EOF
