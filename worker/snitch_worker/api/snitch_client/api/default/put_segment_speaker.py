from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.segment_conflict import SegmentConflict
from ...models.segment_speaker import SegmentSpeaker
from ...models.segment_speaker_input import SegmentSpeakerInput
from typing import cast



def _get_kwargs(
    id: int,
    index: int,
    *,
    body: SegmentSpeakerInput,
    origin: str,
    x_csrf_token: str,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    headers["Origin"] = origin

    headers["X-CSRF-Token"] = x_csrf_token







    _kwargs: dict[str, Any] = {
        "method": "put",
        "url": "/api/clips/{id}/segments/{index}/speaker".format(id=quote(str(id), safe=""),index=quote(str(index), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | SegmentConflict | SegmentSpeaker:
    if response.status_code == 200:
        response_200 = SegmentSpeaker.from_dict(response.json())



        return response_200

    if response.status_code == 409:
        response_409 = SegmentConflict.from_dict(response.json())



        return response_409

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | SegmentConflict | SegmentSpeaker]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    id: int,
    index: int,
    *,
    client: AuthenticatedClient,
    body: SegmentSpeakerInput,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | SegmentConflict | SegmentSpeaker]:
    """  Utilisateurs connectés, pour un clip visible. Désigne le Locuteur local d'un segment parmi ceux du
    même clip et de la même piste audio, ou revient au locuteur déduit des tours de parole
    (local_speaker_id null). Ne modifie ni les tours de parole, ni les empreintes, ni les attributions
    aux Personnes, ni les échantillons vocaux, et ne déclenche ni diarisation ni réindexation
    sémantique. La correction est historisée (type speaker) ; l'affichage du segment, la recherche et le
    filtre par Personne utilisent le locuteur corrigé.

    Args:
        id (int):
        index (int):
        origin (str):
        x_csrf_token (str):
        body (SegmentSpeakerInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | SegmentConflict | SegmentSpeaker]
     """


    kwargs = _get_kwargs(
        id=id,
index=index,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    id: int,
    index: int,
    *,
    client: AuthenticatedClient,
    body: SegmentSpeakerInput,
    origin: str,
    x_csrf_token: str,

) -> Error | SegmentConflict | SegmentSpeaker | None:
    """  Utilisateurs connectés, pour un clip visible. Désigne le Locuteur local d'un segment parmi ceux du
    même clip et de la même piste audio, ou revient au locuteur déduit des tours de parole
    (local_speaker_id null). Ne modifie ni les tours de parole, ni les empreintes, ni les attributions
    aux Personnes, ni les échantillons vocaux, et ne déclenche ni diarisation ni réindexation
    sémantique. La correction est historisée (type speaker) ; l'affichage du segment, la recherche et le
    filtre par Personne utilisent le locuteur corrigé.

    Args:
        id (int):
        index (int):
        origin (str):
        x_csrf_token (str):
        body (SegmentSpeakerInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | SegmentConflict | SegmentSpeaker
     """


    return sync_detailed(
        id=id,
index=index,
client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    ).parsed

async def asyncio_detailed(
    id: int,
    index: int,
    *,
    client: AuthenticatedClient,
    body: SegmentSpeakerInput,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | SegmentConflict | SegmentSpeaker]:
    """  Utilisateurs connectés, pour un clip visible. Désigne le Locuteur local d'un segment parmi ceux du
    même clip et de la même piste audio, ou revient au locuteur déduit des tours de parole
    (local_speaker_id null). Ne modifie ni les tours de parole, ni les empreintes, ni les attributions
    aux Personnes, ni les échantillons vocaux, et ne déclenche ni diarisation ni réindexation
    sémantique. La correction est historisée (type speaker) ; l'affichage du segment, la recherche et le
    filtre par Personne utilisent le locuteur corrigé.

    Args:
        id (int):
        index (int):
        origin (str):
        x_csrf_token (str):
        body (SegmentSpeakerInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | SegmentConflict | SegmentSpeaker]
     """


    kwargs = _get_kwargs(
        id=id,
index=index,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    id: int,
    index: int,
    *,
    client: AuthenticatedClient,
    body: SegmentSpeakerInput,
    origin: str,
    x_csrf_token: str,

) -> Error | SegmentConflict | SegmentSpeaker | None:
    """  Utilisateurs connectés, pour un clip visible. Désigne le Locuteur local d'un segment parmi ceux du
    même clip et de la même piste audio, ou revient au locuteur déduit des tours de parole
    (local_speaker_id null). Ne modifie ni les tours de parole, ni les empreintes, ni les attributions
    aux Personnes, ni les échantillons vocaux, et ne déclenche ni diarisation ni réindexation
    sémantique. La correction est historisée (type speaker) ; l'affichage du segment, la recherche et le
    filtre par Personne utilisent le locuteur corrigé.

    Args:
        id (int):
        index (int):
        origin (str):
        x_csrf_token (str):
        body (SegmentSpeakerInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | SegmentConflict | SegmentSpeaker
     """


    return (await asyncio_detailed(
        id=id,
index=index,
client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )).parsed
