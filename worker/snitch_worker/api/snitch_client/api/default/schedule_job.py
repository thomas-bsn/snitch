from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.corrections_warning import CorrectionsWarning
from ...models.error import Error
from ...models.job import Job
from ...models.job_input import JobInput
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    *,
    body: JobInput,
    confirm_replace_corrections: bool | Unset = UNSET,
    origin: str,
    x_csrf_token: str,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    headers["Origin"] = origin

    headers["X-CSRF-Token"] = x_csrf_token





    params: dict[str, Any] = {}

    params["confirm_replace_corrections"] = confirm_replace_corrections


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/admin/jobs",
        "params": params,
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> CorrectionsWarning | Error | Job:
    if response.status_code == 200:
        response_200 = Job.from_dict(response.json())



        return response_200

    if response.status_code == 201:
        response_201 = Job.from_dict(response.json())



        return response_201

    if response.status_code == 409:
        response_409 = CorrectionsWarning.from_dict(response.json())



        return response_409

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[CorrectionsWarning | Error | Job]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: JobInput,
    confirm_replace_corrections: bool | Unset = UNSET,
    origin: str,
    x_csrf_token: str,

) -> Response[CorrectionsWarning | Error | Job]:
    """  Administrateurs uniquement. La planification répétée d'une tâche active est idempotente.

    Args:
        confirm_replace_corrections (bool | Unset):
        origin (str):
        x_csrf_token (str):
        body (JobInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[CorrectionsWarning | Error | Job]
     """


    kwargs = _get_kwargs(
        body=body,
confirm_replace_corrections=confirm_replace_corrections,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    *,
    client: AuthenticatedClient | Client,
    body: JobInput,
    confirm_replace_corrections: bool | Unset = UNSET,
    origin: str,
    x_csrf_token: str,

) -> CorrectionsWarning | Error | Job | None:
    """  Administrateurs uniquement. La planification répétée d'une tâche active est idempotente.

    Args:
        confirm_replace_corrections (bool | Unset):
        origin (str):
        x_csrf_token (str):
        body (JobInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        CorrectionsWarning | Error | Job
     """


    return sync_detailed(
        client=client,
body=body,
confirm_replace_corrections=confirm_replace_corrections,
origin=origin,
x_csrf_token=x_csrf_token,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: JobInput,
    confirm_replace_corrections: bool | Unset = UNSET,
    origin: str,
    x_csrf_token: str,

) -> Response[CorrectionsWarning | Error | Job]:
    """  Administrateurs uniquement. La planification répétée d'une tâche active est idempotente.

    Args:
        confirm_replace_corrections (bool | Unset):
        origin (str):
        x_csrf_token (str):
        body (JobInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[CorrectionsWarning | Error | Job]
     """


    kwargs = _get_kwargs(
        body=body,
confirm_replace_corrections=confirm_replace_corrections,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    body: JobInput,
    confirm_replace_corrections: bool | Unset = UNSET,
    origin: str,
    x_csrf_token: str,

) -> CorrectionsWarning | Error | Job | None:
    """  Administrateurs uniquement. La planification répétée d'une tâche active est idempotente.

    Args:
        confirm_replace_corrections (bool | Unset):
        origin (str):
        x_csrf_token (str):
        body (JobInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        CorrectionsWarning | Error | Job
     """


    return (await asyncio_detailed(
        client=client,
body=body,
confirm_replace_corrections=confirm_replace_corrections,
origin=origin,
x_csrf_token=x_csrf_token,

    )).parsed
