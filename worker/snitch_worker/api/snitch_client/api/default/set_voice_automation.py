from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.voice_automation_input import VoiceAutomationInput
from ...models.voice_automation_result import VoiceAutomationResult
from typing import cast



def _get_kwargs(
    *,
    body: VoiceAutomationInput,
    origin: str,
    x_csrf_token: str,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    headers["Origin"] = origin

    headers["X-CSRF-Token"] = x_csrf_token







    _kwargs: dict[str, Any] = {
        "method": "put",
        "url": "/api/admin/voice-automation",
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | VoiceAutomationResult:
    if response.status_code == 200:
        response_200 = VoiceAutomationResult.from_dict(response.json())



        return response_200

    if response.status_code == 409:
        response_409 = Error.from_dict(response.json())



        return response_409

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | VoiceAutomationResult]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient,
    body: VoiceAutomationInput,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | VoiceAutomationResult]:
    """  Administrateurs uniquement : seul endpoint qui active ou désactive l'attribution automatique.
    L'activation sous les critères exige confirm_unmet_criteria. Chaque changement effectif est tracé
    avec la mesure du moment. La désactivation ramène les attributions automatiques à des suggestions ;
    aucune référence n'est créée ni modifiée.

    Args:
        origin (str):
        x_csrf_token (str):
        body (VoiceAutomationInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | VoiceAutomationResult]
     """


    kwargs = _get_kwargs(
        body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    *,
    client: AuthenticatedClient,
    body: VoiceAutomationInput,
    origin: str,
    x_csrf_token: str,

) -> Error | VoiceAutomationResult | None:
    """  Administrateurs uniquement : seul endpoint qui active ou désactive l'attribution automatique.
    L'activation sous les critères exige confirm_unmet_criteria. Chaque changement effectif est tracé
    avec la mesure du moment. La désactivation ramène les attributions automatiques à des suggestions ;
    aucune référence n'est créée ni modifiée.

    Args:
        origin (str):
        x_csrf_token (str):
        body (VoiceAutomationInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | VoiceAutomationResult
     """


    return sync_detailed(
        client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient,
    body: VoiceAutomationInput,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | VoiceAutomationResult]:
    """  Administrateurs uniquement : seul endpoint qui active ou désactive l'attribution automatique.
    L'activation sous les critères exige confirm_unmet_criteria. Chaque changement effectif est tracé
    avec la mesure du moment. La désactivation ramène les attributions automatiques à des suggestions ;
    aucune référence n'est créée ni modifiée.

    Args:
        origin (str):
        x_csrf_token (str):
        body (VoiceAutomationInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | VoiceAutomationResult]
     """


    kwargs = _get_kwargs(
        body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    *,
    client: AuthenticatedClient,
    body: VoiceAutomationInput,
    origin: str,
    x_csrf_token: str,

) -> Error | VoiceAutomationResult | None:
    """  Administrateurs uniquement : seul endpoint qui active ou désactive l'attribution automatique.
    L'activation sous les critères exige confirm_unmet_criteria. Chaque changement effectif est tracé
    avec la mesure du moment. La désactivation ramène les attributions automatiques à des suggestions ;
    aucune référence n'est créée ni modifiée.

    Args:
        origin (str):
        x_csrf_token (str):
        body (VoiceAutomationInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | VoiceAutomationResult
     """


    return (await asyncio_detailed(
        client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )).parsed
