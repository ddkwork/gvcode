export GOOS=ios
export GOARCH=arm64 # Or your target architecture like 'amd64' for simulator
export CGO_ENABLED=1
export CC=$(xcrun --sdk iphoneos --find clang) # For device; use 'iphonesimulator' for simulator
SDK_PATH=$(xcrun --sdk iphoneos --show-sdk-path) # For device; use 'iphonesimulator' for simulator

# Set your desired minimum iOS version here (e.g., 15.0)
MIN_IOS_VERSION="15.0" 

export CGO_CFLAGS="-fembed-bitcode -isysroot $SDK_PATH -arch arm64 -mios-version-min=$MIN_IOS_VERSION"
export CGO_LDFLAGS="-fembed-bitcode -isysroot $SDK_PATH -arch arm64 -mios-version-min=$MIN_IOS_VERSION"

# Now run your gogio command or go build command
# For example:
# gogio -x -target ios -appid ws.looz.fern github.com/oligo/gvcode
# or for a manual build of your main package:
# go build -v -x -o ./my-app-output .
gogio -x -target ios  -appid ws.looz.fern \
    -signkey 'Apple Distribution: Zhijian Zhang (6YHUCQ637C)' \
    -notaryid 'zhangzj33@gmail.com' \
    -notarypass 'aoyh-oezh-dvtz-kzld' \
    -notaryteamid '6YHUCQ637C' ./