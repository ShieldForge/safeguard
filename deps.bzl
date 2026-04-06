load("@bazel_gazelle//:deps.bzl", "go_repository")

def go_dependencies():
    go_repository(
        name = "com_github_rs_zerolog",
        importpath = "github.com/rs/zerolog",
        sum = "h1:Ql6K6qYHEzB6xvu4+AU0BoRoqf9vFPcc4o7MUIdPW8Y=",
        version = "v1.34.0",
    )
    go_repository(
        name = "com_github_pkg_browser",
        importpath = "github.com/pkg/browser",
        sum = "h1:+mdjkGKdHQG3305AYmdv1U2eRNDiU2ErMBj1gwrq8eQ=",
        version = "v0.0.0-20240102092130-5ac0b6a4141c",
    )
    go_repository(
        name = "com_github_winfsp_cgofuse",
        importpath = "github.com/winfsp/cgofuse",
        sum = "h1:3kIHsoL3O0VEzNA3Xzn+x+KxCpKLhhdcSnP6LlZBFLg=",
        version = "v1.6.0",
    )
    go_repository(
        name = "com_github_open_policy_agent_opa",
        importpath = "github.com/open-policy-agent/opa",
        sum = "h1:VCuxBUuTk6J+YUClvY90s5g7rlFgjEIkx2Aq/1NmTlc=",
        version = "v1.12.2",
    )
    go_repository(
        name = "com_github_mattn_go_colorable",
        importpath = "github.com/mattn/go-colorable",
        sum = "h1:fFA4WZxdEF4tXPZVKMLwD8oUnCTTo08duU7wxecdEvA=",
        version = "v0.1.13",
    )
    go_repository(
        name = "com_github_mattn_go_isatty",
        importpath = "github.com/mattn/go-isatty",
        sum = "h1:BTarxUcIeDqL27Mc+lolgwaHD3tnZJYU6bT3xo8P2QU=",
        version = "v0.0.19",
    )
    go_repository(
        name = "org_golang_x_sys",
        importpath = "golang.org/x/sys",
        sum = "h1:Af8nKPmuFypiUBjVoU9V20FiaFXOcuZI21p0ycVYYGE=",
        version = "v0.15.0",
    )
