package main

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/codecommit"
	"github.com/aws/aws-sdk-go-v2/service/codecommit/types"
	"github.com/aws/aws-sdk-go-v2/service/iam"
)

type Handle struct {
	iam      *iam.Client
	client   *codecommit.Client
	userName string
	userId   string
	userArn  string

	region     string
	returnMine bool

	prs   []string
	mutex sync.Mutex
}

func (h *Handle) AddPR(repositoryName, pullRequestId string) {
	url := fmt.Sprintf("https://%s.console.aws.amazon.com/codesuite/codecommit/repositories/%s/pull-requests/%s/details?region=%s",
		h.region, repositoryName, pullRequestId, h.region)

	h.mutex.Lock()
	h.prs = append(h.prs, url)
	h.mutex.Unlock()
}

func (h *Handle) checkPullRequest(ctx context.Context, repositoryName, pullRequestId string) error {
	resp, err := h.client.GetPullRequest(ctx, &codecommit.GetPullRequestInput{
		PullRequestId: &pullRequestId,
	})
	if err != nil {
		return err
	}

	isMine := resp.PullRequest.AuthorArn != nil && *resp.PullRequest.AuthorArn == h.userArn

	if h.returnMine {
		// Only return PRs I authored.
		if isMine {
			h.AddPR(repositoryName, pullRequestId)
		}
		return nil
	}

	if isMine {
		// Only return PRs I need to approve.
		return nil
	}

	for _, rule := range resp.PullRequest.ApprovalRules {
		// The approval rule content is just a string of JSON data. A bit
		// hacky, but rather than parse it just check if the user shows up
		// anywhere within it.
		if strings.Contains(*rule.ApprovalRuleContent, "CodeCommitApprovers:"+h.userName) {
			h.AddPR(repositoryName, pullRequestId)
			break
		}
	}

	return nil
}

func (h *Handle) checkRepository(ctx context.Context, repositoryName string) error {
	// Optionally filter by PR author.
	var authorArn *string
	if h.returnMine {
		authorArn = &h.userArn
	}

	resp, err := h.client.ListPullRequests(ctx, &codecommit.ListPullRequestsInput{
		RepositoryName:    &repositoryName,
		PullRequestStatus: types.PullRequestStatusEnumOpen,
		AuthorArn:         authorArn,
	})
	if err != nil {
		return err
	}

	for _, id := range resp.PullRequestIds {
		Debugln("Found PR: ", id)

		err := h.checkPullRequest(ctx, repositoryName, id)
		if err != nil {
			return err
		}
	}

	return nil
}

func (h *Handle) CacheUserName(ctx context.Context) error {
	// Fetch the currently logged in user.
	resp, err := h.iam.GetUser(ctx, &iam.GetUserInput{})
	if err != nil {
		return err
	}

	h.userName = *resp.User.UserName
	h.userId = *resp.User.UserId
	h.userArn = *resp.User.Arn

	return nil
}

func main() {
	var region string
	var returnMine bool

	flag.BoolVar(&DebugMode, "debug", false, "Enable debug logging")
	flag.StringVar(&region, "region", "us-east-2", "AWS region")
	flag.BoolVar(&returnMine, "mine", false, "List open PRs created by me")

	flag.Parse()

	if DebugMode {
		DisableSingleLineMode(0)
	}

	// Load the Shared AWS Configuration (~/.aws/config)
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		Fatalln(err)
	}

	h := Handle{
		iam:        iam.NewFromConfig(cfg),
		client:     codecommit.NewFromConfig(cfg),
		region:     region,
		returnMine: returnMine,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	err = h.CacheUserName(ctx)
	if err != nil {
		Fatalln(err)
	}

	repoResp, err := h.client.ListRepositories(ctx, &codecommit.ListRepositoriesInput{})
	if err != nil {
		Fatalln(err)
	}

	for _, repository := range repoResp.Repositories {
		repository := repository
		Infoln(ctag("Scanning repo:"), " ", ctext(*repository.RepositoryName))

		err := h.checkRepository(ctx, *repository.RepositoryName)
		if err != nil {
			Fatalln(err)
		}
	}

	for _, url := range h.prs {
		Println(url)
	}
	if len(h.prs) <= 0 {
		DisableSingleLineMode(OnDisableClearLine)
	}
}
