# Notary Governance

The following document outlines Notary project governance.

## The Notary Project

The Notary project consists of several repositories known as subprojects that enable community cohorts to experiment and implement solutions across the scope of the project.

## Maintainers Structure

There are two types of maintainers in the Notary project organized hierarchically. Notary org maintainers oversee the overall project and its health. Subproject maintainers focus on a single codebase or a group of related codebases. 

Changes in maintainership have to be announced via an [issue](https://github.com/notaryproject/notaryproject/issues/new).

### Maintainer Responsibility
Notary maintainers adhere to the requirements and responsibilities set forth in the respective [Notary Org Maintainers](#notary-org-maintainers) and [Subproject Maintainers](#subproject-maintainers). They further pledge the following:
* To act in the best interest of the project and subprojects at all times.
* To ensure that project and subproject development and direction is a function of community needs.
* To never take any action while hesitant that it is the right action to take.
* To fulfill the responsibilities outlined in this document and its dependents.

### Notary Org Maintainers

The [Notary Org maintainers](MAINTAINERS) are responsible for:

* Maintaining the mission, vision, values, and scope of the project
* Refining the governance and charter as needed
* Making project level decisions
* Resolving escalated project decisions when the subproject maintainers responsible are blocked
* Managing the Notary brand
* Controlling access to Notary assets such as source repositories, hosting, project calendars
* Deciding what subprojects are part of the Notary project
* Deciding on the creation of new subprojects
* Overseeing the resolution and disclosure of security issues
* Managing financial decisions related to the project

Changes to org maintainers use the following:

* Any subproject maintainer is eligible for a position as an org maintainer
* No one company or organization can employ a simple majority of the org maintainers
* An org maintainer may step down by submitting an [issue](https://github.com/notaryproject/notaryproject/issues/new) stating their intent and they will be moved to emeritus.
* Org maintainers MUST remain active on the project. If they are unresponsive for > 3 months they will lose org maintainership unless a [super-majority](https://en.wikipedia.org/wiki/Supermajority#Two-thirds_vote) of the other org maintainers agrees to extend the period to be greater than 3 months
* Any eligible person may stand as an org maintainer by opening a [PR](https://github.com/notaryproject/notaryproject/pulls).
* When a PR is opened the project maintainers may vote
  * The voting period will be open for a minimum of three business days and will remain open until a super-majority of project maintainers has voted
  * Only current org maintainers are eligible to vote via casting a single vote each via a -1/+1 comment on the nomination issue or approving in GitHub.
  * Once a [super-majority](https://en.wikipedia.org/wiki/Supermajority#Two-thirds_vote) has been reached the maintainer elect must complete [onboarding](#onboarding-a-new-maintainer) prior to becoming an official Notary maintainer.
  * Once the maintainer onboarding has been completed a pull request is made on the repo adding the new maintainer to the [MAINTAINERS](MAINTAINERS) file.
* When an org maintainer steps down, they become an emeritus maintainer

### Subproject Maintainers

Subproject maintainers are responsible for activities surrounding the development and release of content (eg. code, specifications, documentation) or the tasks needed to execute their subproject (e.g., community management) within the designated repository, or repositories associated with the subproject (e.g., community management). Technical decisions for code resides with the subproject maintainers unless there is a decision related to cross maintainer groups that cannot be resolved by those groups. Those cases can be escalated to the org maintainers.

Subprojects may be responsible for one or many repositories.

Subproject maintainers do not need to be software developers. No explicit role is placed upon them and they can be anyone appropriate for the work being produced. For example, if a repository is for documentation it would be appropriate for maintainers to be technical writers.

Changes to maintainers use the following:

* A subproject maintainer may step down by submitting an [issue](https://github.com/notaryproject/notaryproject/issues/new) stating their intent and they will be moved to emeritus.
* Maintainers MUST remain active. If they are unresponsive for > 6 months they will be automatically removed unless a [super-majority](https://en.wikipedia.org/wiki/Supermajority#Two-thirds_vote) of the other subproject maintainers agrees to extend the period to be greater than 6 months
* Potential new maintainers should be ongoing active participants in the project
* New maintainers can be added to a subproject by a [super-majority](https://en.wikipedia.org/wiki/Supermajority#Two-thirds_vote) vote of the existing subproject maintainers
* When a subproject has no maintainers the Notary org maintainers become responsible for it and may archive the subproject or find new maintainers

### Onboarding a New Maintainer
New Notary maintainers participate in an onboarding period during which they fulfill all code review and issue management responsibilities that are required for their role. The length of this onboarding period is variable, and is considered complete once both the existing maintainers and the candidate maintainer are comfortable with the candidate's competency in the responsibilities of maintainership. This process MUST be completed prior to the candidate being named an official Notary maintainer.

The onboarding period is intended to ensure that the to-be-appointed maintainer is able/willing to take on the time requirements, familiar with core logic and concepts, understands the overall system architecture and interactions that comprise it, and is able to work well with both the existing maintainers and the community.

## Decision Making at the Notary org level

When maintainers need to make decisions there are two ways decisions are made, unless described elsewhere.

The default decision making process is [lazy-consensus](http://communitymgt.wikia.com/wiki/Lazy_consensus). This means that any decision is considered supported by the team making it as long as no one objects. Silence on any consensus decision is implicit agreement and equivalent to explicit agreement. Explicit agreement may be stated at will. In the case of security critical decisions more explicit consensus may be needed.

When a consensus cannot be found a maintainer can call for a [majority](https://en.wikipedia.org/wiki/Majority) vote on a decision.

Many of the day-to-day project maintenance can be done by a lazy consensus model. But the following items must be called to vote:

* Removing a maintainer for any reason other than inactivity (super majority)
* Changing the governance rules (this document) (super majority)
* Licensing and intellectual property changes (including new logos, wordmarks) (simple majority)
* Adding, archiving, or removing subprojects (simple majority)
* Utilizing Notary/CNCF money for anything CNCF deems "not cheap and easy" (simple majority)

New subprojects should be created (or added) with a well defined mission and goals, and significant changes should be voted on by both the subproject maintainers and the org maintainers.

Other decisions may, but do not need to be, called out and put up for decision via creating an [issue](https://github.com/notaryproject/notaryproject/issues/new) at any time and by anyone. By default, any decisions called to a vote will be for a _simple majority_ vote.

Meetings should be publically documented (Slack, CNCF calendar etc), and recorded and notes kept. Meetings should have a chair, this is a rotating role not restricted to maintainers.

## Code of Conduct

This Notary project has adopted the [CNCF Code of Conduct](https://github.com/cncf/foundation/blob/master/code-of-conduct.md).

## Attributions

* This governance model was created using both the [SPIFFE](https://github.com/spiffe/spire/blob/main/MAINTAINERS.md) and [Helm](https://github.com/helm/community/blob/main/governance/governance.md) governance documents.

## DCO and Licenses

The following licenses and contributor agreements will be used for Notary projects:

* [Apache 2.0](https://opensource.org/licenses/Apache-2.0) for code
* [Developer Certificate of Origin](https://developercertificate.org/) for new contributions
