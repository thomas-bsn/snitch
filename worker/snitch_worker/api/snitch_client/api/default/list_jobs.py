from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.job_page import JobPage
from ...models.list_jobs_status import ListJobsStatus
from ...models.list_jobs_workflow import ListJobsWorkflow
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    *,
    status: ListJobsStatus | Unset = UNSET,
    clip_id: int | Unset = UNSET,
    step: str | Unset = UNSET,
    page: int | Unset = 1,
    page_size: int | Unset = 24,
    workflow: ListJobsWorkflow | Unset = UNSET,

) -> dict[str, Any]:




    params: dict[str, Any] = {}

    json_status: str | Unset = UNSET
    if not isinstance(status, Unset):
        json_status = status.value

    params["status"] = json_status

    params["clip_id"] = clip_id

    params["step"] = step

    params["page"] = page

    params["page_size"] = page_size

    json_workflow: str | Unset = UNSET
    if not isinstance(workflow, Unset):
        json_workflow = workflow.value

    params["workflow"] = json_workflow


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/admin/jobs",
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | JobPage:
    if response.status_code == 200:
        response_200 = JobPage.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | JobPage]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    status: ListJobsStatus | Unset = UNSET,
    clip_id: int | Unset = UNSET,
    step: str | Unset = UNSET,
    page: int | Unset = 1,
    page_size: int | Unset = 24,
    workflow: ListJobsWorkflow | Unset = UNSET,

) -> Response[Error | JobPage]:
    """  Administrateurs uniquement. Aucun chemin local n'est exposé.

    Args:
        status (ListJobsStatus | Unset):
        clip_id (int | Unset):
        step (str | Unset):
        page (int | Unset):  Default: 1.
        page_size (int | Unset):  Default: 24.
        workflow (ListJobsWorkflow | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | JobPage]
     """


    kwargs = _get_kwargs(
        status=status,
clip_id=clip_id,
step=step,
page=page,
page_size=page_size,
workflow=workflow,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    *,
    client: AuthenticatedClient | Client,
    status: ListJobsStatus | Unset = UNSET,
    clip_id: int | Unset = UNSET,
    step: str | Unset = UNSET,
    page: int | Unset = 1,
    page_size: int | Unset = 24,
    workflow: ListJobsWorkflow | Unset = UNSET,

) -> Error | JobPage | None:
    """  Administrateurs uniquement. Aucun chemin local n'est exposé.

    Args:
        status (ListJobsStatus | Unset):
        clip_id (int | Unset):
        step (str | Unset):
        page (int | Unset):  Default: 1.
        page_size (int | Unset):  Default: 24.
        workflow (ListJobsWorkflow | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | JobPage
     """


    return sync_detailed(
        client=client,
status=status,
clip_id=clip_id,
step=step,
page=page,
page_size=page_size,
workflow=workflow,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    status: ListJobsStatus | Unset = UNSET,
    clip_id: int | Unset = UNSET,
    step: str | Unset = UNSET,
    page: int | Unset = 1,
    page_size: int | Unset = 24,
    workflow: ListJobsWorkflow | Unset = UNSET,

) -> Response[Error | JobPage]:
    """  Administrateurs uniquement. Aucun chemin local n'est exposé.

    Args:
        status (ListJobsStatus | Unset):
        clip_id (int | Unset):
        step (str | Unset):
        page (int | Unset):  Default: 1.
        page_size (int | Unset):  Default: 24.
        workflow (ListJobsWorkflow | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | JobPage]
     """


    kwargs = _get_kwargs(
        status=status,
clip_id=clip_id,
step=step,
page=page,
page_size=page_size,
workflow=workflow,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    status: ListJobsStatus | Unset = UNSET,
    clip_id: int | Unset = UNSET,
    step: str | Unset = UNSET,
    page: int | Unset = 1,
    page_size: int | Unset = 24,
    workflow: ListJobsWorkflow | Unset = UNSET,

) -> Error | JobPage | None:
    """  Administrateurs uniquement. Aucun chemin local n'est exposé.

    Args:
        status (ListJobsStatus | Unset):
        clip_id (int | Unset):
        step (str | Unset):
        page (int | Unset):  Default: 1.
        page_size (int | Unset):  Default: 24.
        workflow (ListJobsWorkflow | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | JobPage
     """


    return (await asyncio_detailed(
        client=client,
status=status,
clip_id=clip_id,
step=step,
page=page,
page_size=page_size,
workflow=workflow,

    )).parsed
